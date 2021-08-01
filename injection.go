/*
 * Copyright (c) 2021 boot-go
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package boot

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// DependencyInjectionError contains a detail description for the cause of the injection failure
type DependencyInjectionError struct {
	err    string
	detail string
}

const (
	fieldTag          = "boot" // this key should follow the package name
	fieldTagConfig    = "config"
	fieldTagWire      = "wire"
	fieldTagName      = "name"
	fieldTagWireEnv   = "env"
	fieldTagWirePanic = "panic"
)

func (e *DependencyInjectionError) Error() string {
	return fmt.Sprintf("Error %s %s", e.err, e.detail)
}

func resolveDependency(resolveEntry *entry, registry *registry) (entries []*entry, err error) {
	// exit if this component is already initialized
	if resolveEntry.state != Created {
		return entries, nil
	}
	Logger.Debug.Printf("resolving dependencies for %s", resolveEntry.getFullName())
	componentValue := reflect.ValueOf(resolveEntry.component)
	if componentValue.Kind() == reflect.Ptr {
		componentValue = componentValue.Elem()
	}
	for j := 0; j < componentValue.Type().NumField(); j++ {
		field := componentValue.Type().Field(j)
		fieldValue := componentValue.Field(j)
		if tag, ok := field.Tag.Lookup(fieldTag); ok {
			parsedTag, ok := parseStructTag(tag)
			if !ok {
				return nil, &DependencyInjectionError{
					err: "field contains unparsable tag",
					detail: " <" + componentValue.Type().Name() + "." + field.Name +
						" `" + tag + "`>",
				}
			}
			switch parsedTag.name {
			case fieldTagWire:
				name := parsedTag.options[fieldTagName]
				if name == "" {
					name = DefaultName
				}
				if resolvedEntries, err := processWiring(name, field, componentValue, fieldValue, registry); err == nil {
					entries = append(entries, resolvedEntries...)
				} else {
					return nil, err
				}
			case fieldTagConfig:
				if err := processConfiguration(field, componentValue, fieldValue, parsedTag); err != nil {
					return nil, err
				}
			default:
				return nil, &DependencyInjectionError{
					err: "dependency field has unsupported tag",
					detail: " <" + componentValue.Type().Name() + "." + field.Name +
						" `" + tag + "`>",
				}
			}
		}
	}
	// initialize component
	Logger.Debug.Printf("initializing %s", resolveEntry.getFullName())
	err = initComponent(resolveEntry)
	if err != nil {
		return
	}
	resolveEntry.state = Initialized
	entries = append(entries, resolveEntry)
	return entries, nil
}

func initComponent(resolveEntry *entry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = errors.New("initializing " + resolveEntry.getFullName() + " failed with error: " + v.Error())
			case string:
				err = errors.New("initializing " + resolveEntry.getFullName() + " failed with message: " + v)
			default:
				err = errors.New("initializing " + resolveEntry.getFullName() + " failed")
			}
		}
	}()
	resolveEntry.component.Init()
	return nil
}

func processWiring(name string, field reflect.StructField, componentValue reflect.Value, fieldValue reflect.Value, registry *registry) ([]*entry, error) {
	if fieldValue.Kind() != reflect.Ptr && fieldValue.Kind() != reflect.Interface {
		return nil, &DependencyInjectionError{
			err:    "dependency field is not a pointer receiver",
			detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
		}
	}
	var matchingValues []reflect.Value
	for _, list := range registry.entries {
		e := list[name]
		if e != nil {
			controlValue := reflect.ValueOf(e.component)
			if controlValue.Type().AssignableTo(field.Type) {
				if fieldValue.CanSet() {
					matchingValues = append(matchingValues, controlValue)
				} else {
					return nil, &DependencyInjectionError{
						err:    "dependency value cannot be set into",
						detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
					}
				}
			}
		}
	}
	if len(matchingValues) == 1 {
		typeName := matchingValues[0].Elem().Type().PkgPath() + "/" + matchingValues[0].Elem().Type().Name()
		e := registry.entries[typeName][name]
		if e.state == Created {
			entries, err := resolveDependency(e, registry)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(e.component))
			return entries, nil
		}
		fieldValue.Set(reflect.ValueOf(e.component))
	} else if len(matchingValues) == 0 {
		return nil, &DependencyInjectionError{
			err:    "dependency value not found for",
			detail: "<" + name + ":" + componentValue.Type().Name() + "." + field.Name + ">",
		}
	} else {
		return nil, &DependencyInjectionError{
			err:    "multiple dependency values found for",
			detail: "<" + name + ":" + componentValue.Type().Name() + "." + field.Name + ">",
		}
	}
	return []*entry{}, nil // this
}

func processConfiguration(field reflect.StructField, componentValue reflect.Value, fieldValue reflect.Value, tag *tag) error {
	panicOnFail := false
	if tag.hasOption(fieldTagWirePanic) {
		panicOnFail = true
	}
	if tag.hasOption(fieldTagWireEnv) {
		if cfg := tag.options[fieldTagWireEnv]; len(cfg) > 3 {
			cfg = cfg[2 : len(cfg)-1]
			if cfgValue, ok := os.LookupEnv(cfg); ok {
				if fieldValue.CanSet() {
					processConfigString(field, fieldValue, cfgValue, cfg)
					err := processConfigInt(field, componentValue, fieldValue, cfgValue, panicOnFail, cfg)
					if err != nil {
						return err
					}
					err = processConfigBool(field, componentValue, fieldValue, cfgValue, panicOnFail, cfg)
					if err != nil {
						return err
					}
				}
			} else {
				if panicOnFail {
					return &DependencyInjectionError{
						err:    "failed to load configuration value for " + cfg,
						detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
					}
				}
				Logger.Warn.Printf("failed to parse configuration value %s for %s\n", cfgValue, "<"+componentValue.Type().Name()+"."+field.Name+">")
			}
		} else {
			return fmt.Errorf("unsupported env value %s", cfg)
		}
	} else {
		return &DependencyInjectionError{
			err:    "unsupported configuration options found",
			detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
		}
	}
	return nil
}

func processConfigBool(field reflect.StructField, componentValue reflect.Value, fieldValue reflect.Value, cfgValue string, panicOnFail bool, cfg string) error {
	if field.Type.Name() == "bool" {
		if fieldValue.Bool() == false {
			boolValue, err := strconv.ParseBool(cfgValue)
			if err != nil {
				if panicOnFail {
					return &DependencyInjectionError{
						err:    "failed to load configuration value for " + cfg,
						detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
					}
				}
				Logger.Warn.Printf("failed to parse configuration value %s as boolean: %s\n", cfgValue, err)
			}
			fieldValue.SetBool(boolValue)
			Logger.Debug.Printf("setting boolean configuration %s=%s\n", cfg, cfgValue)
		}
	}
	return nil
}

func processConfigInt(field reflect.StructField, componentValue reflect.Value, fieldValue reflect.Value, cfgValue string, panicOnFail bool, cfg string) error {
	if field.Type.Name() == "int" {
		if fieldValue.Int() == 0 {
			intValue, err := strconv.ParseInt(cfgValue, 10, 64)
			if err != nil {
				if panicOnFail {
					return &DependencyInjectionError{
						err:    "failed to load configuration value for " + cfg,
						detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
					}
				}
				Logger.Warn.Printf("failed to parse configuration value %s as integer: %s\n", cfgValue, err)
			}
			fieldValue.SetInt(intValue)
			Logger.Debug.Printf("setting integer configuration  %s=%s\n", cfg, cfgValue)
		}
	}
	return nil
}

func processConfigString(field reflect.StructField, fieldValue reflect.Value, cfgValue string, cfg string) {
	if field.Type.Name() == "string" {
		if fieldValue.String() == "" {
			fieldValue.SetString(cfgValue)
			Logger.Debug.Printf("setting string configuration %s=%s\n", cfg, cfgValue)
		}
	}
}

type tag struct {
	name    string
	options map[string]string
}

func (t *tag) hasOption(name string) bool {
	_, ok := t.options[name]
	return ok
}

func parseStructTag(tagValue string) (*tag, bool) {
	options := make(map[string]string)
	tokens := strings.Split(tagValue, ",")
	name := strings.TrimSpace(tokens[0])
	for i, token := range tokens {
		if i > 0 {
			subtokens := strings.Split(token, ":")
			if len(subtokens) > 0 && len(subtokens) < 3 {
				key := subtokens[0]
				if len(subtokens) == 2 {
					value := subtokens[1]
					if value != "" {
						options[key] = value
					} else {
						return nil, false
					}
				} else {
					options[key] = ""
				}
			} else {
				return nil, false
			}
		}
	}
	return &tag{
		name:    name,
		options: options,
	}, true
}
