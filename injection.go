/*
 * Copyright (c) 2021-2022 boot-go
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
	error
	detail string
}

const (
	fieldTag            = "boot" // this key should follow the package name
	fieldTagConfig      = "config"
	fieldTagWire        = "wire"
	fieldTagName        = "name"
	fieldTagWireKey     = "key"
	fieldTagWirePanic   = "panic"
	fieldTagWireDefault = "default"
)

func (e *DependencyInjectionError) Error() string {
	return fmt.Sprintf("Error %s %s", e.error.Error(), e.detail)
}

func resolveDependency(regEntry *componentManager, reg *registry) (entries []*componentManager, err error) {
	// exit if this component is already initialized
	if regEntry.state != Created {
		return entries, nil
	}
	Logger.Debug.Printf("resolving dependencies for %s", regEntry.getFullName())
	reflectedComponent := reflect.ValueOf(regEntry.component)
	if reflectedComponent.Kind() == reflect.Ptr {
		reflectedComponent = reflectedComponent.Elem()
	}
	for j := 0; j < reflectedComponent.Type().NumField(); j++ {
		field := reflectedComponent.Type().Field(j)
		fieldValue := reflectedComponent.Field(j)
		if tag, ok := field.Tag.Lookup(fieldTag); ok {
			parsedTag, ok := parseStructTag(tag)
			if !ok {
				return nil, &DependencyInjectionError{
					error: errors.New("field contains unparsable tag"),
					detail: " <" + reflectedComponent.Type().Name() + "." + field.Name +
						" `" + tag + "`>",
				}
			}
			switch parsedTag.name {
			case fieldTagWire:
				regEntryName := parsedTag.options[fieldTagName]
				if regEntryName == "" {
					regEntryName = DefaultName
				}
				if resolvedEntries, err := processWiring(reg, reflectedComponent, field, fieldValue, regEntryName); err == nil {
					entries = append(entries, resolvedEntries...)
				} else {
					return nil, err
				}
			case fieldTagConfig:
				if err := processConfiguration(reflectedComponent, field, fieldValue, parsedTag); err != nil {
					return nil, err
				}
			default:
				return nil, &DependencyInjectionError{
					error: errors.New("dependency field has unsupported tag"),
					detail: " <" + reflectedComponent.Type().Name() + "." + field.Name +
						" `" + tag + "`>",
				}
			}
		}
	}
	// initialize component
	Logger.Debug.Printf("initializing %s\n", regEntry.getFullName())
	err = initComponent(regEntry)
	if err != nil {
		regEntry.state = Failed
		return
	}
	regEntry.state = Initialized
	entries = append(entries, regEntry)
	return entries, nil
}

func initComponent(resolveEntry *componentManager) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = errors.New("initializing " + resolveEntry.getFullName() + " panicked with error: " + v.Error())
			case string:
				err = errors.New("initializing " + resolveEntry.getFullName() + " panicked with message: " + v)
			default:
				err = errors.New("initializing " + resolveEntry.getFullName() + " panicked")
			}
		}
	}()
	err = resolveEntry.component.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize component %s - reason: %w", resolveEntry.getFullName(), err)
	}
	return
}

func processWiring(reg *registry, reflectedComponent reflect.Value, field reflect.StructField, fieldValue reflect.Value, regEntryName string) ([]*componentManager, error) {
	if fieldValue.Kind() != reflect.Ptr && fieldValue.Kind() != reflect.Interface {
		return nil, &DependencyInjectionError{
			error:  errors.New("dependency field is not a pointer receiver"),
			detail: "<" + reflectedComponent.Type().Name() + "." + field.Name + ">",
		}
	}
	var matchingValues []reflect.Value
	for _, list := range reg.items {
		e := list[regEntryName]
		if e != nil {
			controlValue := reflect.ValueOf(e.component)
			if controlValue.Type().AssignableTo(field.Type) {
				if fieldValue.CanSet() {
					matchingValues = append(matchingValues, controlValue)
				} else {
					return nil, &DependencyInjectionError{
						error:  errors.New("dependency value cannot be set into"),
						detail: "<" + reflectedComponent.Type().Name() + "." + field.Name + ">",
					}
				}
			}
		}
	}
	switch len(matchingValues) {
	case 1:
		typeName := matchingValues[0].Elem().Type().PkgPath() + "/" + matchingValues[0].Elem().Type().Name()
		e := reg.items[typeName][regEntryName]
		if e.state == Created {
			entries, err := resolveDependency(e, reg)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(e.component))
			return entries, nil
		}
		fieldValue.Set(reflect.ValueOf(e.component))
	case 0:
		return nil, &DependencyInjectionError{
			error:  errors.New("dependency value not found for"),
			detail: "<" + regEntryName + ":" + reflectedComponent.Type().Name() + "." + field.Name + ">",
		}
	default:
		return nil, &DependencyInjectionError{
			error:  errors.New("multiple dependency values found for"),
			detail: "<" + regEntryName + ":" + reflectedComponent.Type().Name() + "." + field.Name + ">",
		}
	}
	return []*componentManager{}, nil // this
}

func processConfiguration(reflectedComponent reflect.Value, field reflect.StructField, fieldValue reflect.Value, tag *tag) error {
	panicOnFail := false
	defaultCfg := ""
	hasDefault := false
	if tag.hasOption(fieldTagWirePanic) {
		panicOnFail = true
	}
	if tag.hasOption(fieldTagWireDefault) {
		defaultCfg = tag.options[fieldTagWireDefault]
		hasDefault = true
	}
	if tag.hasOption(fieldTagWireKey) {
		if cfgKey := tag.options[fieldTagWireKey]; len(cfgKey) > 0 {
			if cfgValue, ok := getConfig(cfgKey); ok || hasDefault {
				if !ok && hasDefault {
					cfgValue = defaultCfg
				}
				if fieldValue.CanSet() {
					err := processConfigValue(reflectedComponent, field, fieldValue, cfgValue, cfgKey, panicOnFail)
					if err != nil {
						return err
					}
				}
			} else {
				if panicOnFail {
					return &DependencyInjectionError{
						error:  errors.New("failed to load configuration value for " + cfgKey),
						detail: "<" + reflectedComponent.Type().Name() + "." + field.Name + ">",
					}
				}
				Logger.Warn.Printf("failed to parse configuration value %s for %s\n", cfgValue, "<"+reflectedComponent.Type().Name()+"."+field.Name+">")
			}
		} else {
			return fmt.Errorf("unsupported tag value %s", cfgKey)
		}
	} else {
		return &DependencyInjectionError{
			error:  errors.New("unsupported configuration options found"),
			detail: "<" + reflectedComponent.Type().Name() + "." + field.Name + ">",
		}
	}
	return nil
}

func processConfigValue(reflectedComponent reflect.Value, field reflect.StructField, fieldValue reflect.Value, cfgValue string, cfgKey string, panicOnFail bool) error {
	processConfigString(field, fieldValue, cfgValue, cfgKey)
	err := processConfigInt(field, reflectedComponent, fieldValue, cfgValue, panicOnFail, cfgKey)
	if err != nil {
		return err
	}
	err = processConfigBool(field, reflectedComponent, fieldValue, cfgValue, panicOnFail, cfgKey)
	if err != nil {
		return err
	}
	return nil
}

func getConfig(cfgKey string) (string, bool) {
	key := ""
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--") && len(arg) > 2 { //nolint:gocritic // using switch is unsuitebale
			key = arg[2:]
		} else if key != "" {
			if cfgKey == key {
				return arg, true
			}
		} else {
			key = ""
		}
	}
	return os.LookupEnv(cfgKey)
}

func processConfigBool(field reflect.StructField, componentValue reflect.Value, fieldValue reflect.Value, cfgValue string, panicOnFail bool, cfg string) error {
	if field.Type.Name() == "bool" {
		if !fieldValue.Bool() {
			boolValue, err := strconv.ParseBool(cfgValue)
			if err != nil {
				if panicOnFail {
					return &DependencyInjectionError{
						error:  errors.New("failed to load configuration value for " + cfg),
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
			const bitSize = 64
			const base = 10
			intValue, err := strconv.ParseInt(cfgValue, base, bitSize)
			if err != nil {
				if panicOnFail {
					return &DependencyInjectionError{
						error:  errors.New("failed to load configuration value for " + cfg),
						detail: "<" + componentValue.Type().Name() + "." + field.Name + ">",
					}
				}
				Logger.Warn.Printf("failed to parse configuration value %s as integer: %s\n", cfgValue, err)
			}
			fieldValue.SetInt(intValue)
			Logger.Debug.Printf("setting integer configuration %s=%s\n", cfg, cfgValue)
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

// parseStructTag returns a reference to a Tag if the tagValue string
// is successfully parsed, which indicated by the second bool return
// value.
func parseStructTag(tagValue string) (*tag, bool) {
	options := make(map[string]string)
	tokens, ok := Split(tagValue, ",", "'")
	if !ok {
		return nil, false
	}
	name := strings.TrimSpace(tokens[0])
	for i, token := range tokens {
		if i > 0 {
			// the split can't fail because the previous split already validates the value
			subtokens, _ := Split(token, ":", "'")
			if len(subtokens) > 0 && len(subtokens) < 3 {
				key := subtokens[0]
				const tokenCut = 2
				if len(subtokens) == tokenCut {
					options[key] = strings.Trim(subtokens[1], " '")
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
