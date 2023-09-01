<div>
    <div align="center"><img src="https://avatars.githubusercontent.com/u/80048065?s=200&u=a95ef12cecad462ed24df9418a8464241301cc16"/></div>
    <div align="center">
        <a href="https://github.com/boot-go/boot/tags"><img alt="Commit tag" src="https://img.shields.io/github/v/tag/boot-go/boot"></a>
        <a href="https://github.com/boot-go/boot/actions/workflows/action.yml"><img src="https://github.com/boot-go/boot/actions/workflows/action.yml/badge.svg?branch=main" alt="github action"></a>
        <a href="https://htmlpreview.github.io/?https://gist.githubusercontent.com/boot-go/c77b22000b3e249510dfb4542847c708/raw/ae9b2e83e9a4adafed6da2160e40855f86ca58a8/cover.html"><img src="https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/boot-go/c77b22000b3e249510dfb4542847c708/raw/test_coverage.json" alt="test coverage"></a>
        <a href="https://goreportcard.com/report/github.com/boot-go/boot"><img src="https://goreportcard.com/badge/github.com/boot-go/boot" alt="go report"></a>
        <a href="https://pkg.go.dev/github.com/boot-go/boot"><img src="https://pkg.go.dev/badge/github.com/boot-go/boot.svg" alt="Go Reference"></a>
        <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
    </div>
</div>

**boot-go** accentuate [component-based development](https://en.wikipedia.org/wiki/Component-based_software_engineering) (CBD).

This is an opinionated view of writing modular and cohesive [Go](https://github.com/golang/go) code. It emphasizes the separation of concerns by loosely coupled components, which communicate with each other via methods and events. The goal is to support writing maintainable code on the long run by leveraging the well-defined standard library.

**boot-go** provided key features are:
- dependency injection
- configuration handling
- code decoupling

### Development characteristic
**boot-go** supports two different development characteristic. For simplicity reason, use the functions ```Register```, ```RegisterName```, ```Override```, ```OverrideName```, ```Shutdown``` and ```Go``` to register components and start **boot-go**. This is the recommended way, despite the fact that one global session is used.

But **boot-go** supports also creating new sessions, so that no global variable is required. In this case, the methods ```Register```, ```RegisterName```, ```Override```, ```OverrideName```, ```Shutdown``` and ```Go``` are provided to register components and start **boot-go**.

### Simple Example
The **hello** component is a very basic example. It contains no fields or provides any interface to interact with other components. The component will just print the _'Hello World'_ message to the console.
```go
package main

import (
	"github.com/boot-go/boot"
	"log"
)

// init() registers a factory method, which creates a hello component.
func init() {
	boot.Register(func() boot.Component {
		return &hello{}
	})
}

// hello is the simplest component.
type hello struct{}

// Init is the initializer of the component.
func (c *hello) Init() error {
	log.Printf("boot-go says > 'Hello World'\n")
	return nil
}

// Start the example and exit after the component was completed.
func main() {
	boot.Go()
}
```

The same example using a new session, which don't need any global variables.
```Go
package main

import (
	"github.com/boot-go/boot"
	"log"
)

// hello is the simplest component.
type hello struct{}

// Init is the initializer of the component.
func (c *hello) Init() error {
	log.Printf("boot-go says > 'Hello World'\n")
	return nil
}

// Start the example and exit after the component was completed.
func main() {
	s := boot.NewSession()
	s.Register(func() boot.Component {
		return &hello{}
	})
	s.Go()
}
```

### Component wiring
This example shows how components get wired automatically with dependency injection. The server component starts at ```:8080``` by default, but the port is configurable by setting the environment variable ```HTTP_SERVER_PORT```. 
```go
package main

import (
	"github.com/boot-go/boot"
	"github.com/boot-go/stack/server/httpcmp"
	"io"
	"net/http"
)

// init() registers a factory method, which creates a hello component
func init() {
	boot.Register(func() boot.Component {
		return &hello{}
	})
}

// hello is a very simple http server example.
// It requires the Eventbus and the chi.Server component. Both components
// are injected by the boot framework automatically
type hello struct {
	Eventbus boot.EventBus  `boot:"wire"`
	Server   httpcmp.Server `boot:"wire"`
}

// Init is the constructor of the component. The handler registration takes place here.
func (h *hello) Init() error {
	// Subscribe to the registration event
	h.Eventbus.Subscribe(func(event httpcmp.InitializedEvent) {
		h.Server.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "boot-go says: 'Hello World'\n")
		})
	})
	return nil
}

// Start the example and test with 'curl localhost:8080'
func main() {
	boot.Go()
}
```

### Component
Everything in **boot-go** starts with a component. They are key fundamental in the development and can be considered as an elementary build block. The essential concept is to get all the necessary components functioning with as less effort as possible. Therefore, components must always provide a default configuration, which uses the most common settings. As an example, a **http server** should always start using port **8080**, unless the developer specifies it. Or a postgres component should try to connect to **localhost:5432** when there is no database url provided.

A component should be _fail tolerant_, _recoverable_, _agnostic_ and _decent_.

| Facet           | Meaning                                       | Example                                                                                                                                                                         |
|-----------------|-----------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| _fail tolerant_ | Don't stop processing on errors.              | A http request can still be processed, even when the metrics server is not available anymore.                                                                                   |
| _recoverable_   | Try to recover from errors.                   | A database component should try to reconnect after losing the connection.                                                                                                       |
| _agnostic_      | Behave the same in any environment.           | A key-value store component should work on a local development machine the same way as in a containerized environment.                                                          |
| _decent_        | Don't overload the developer with complexity. | Keep the interface and events as simple as possible. It's better to build three smaller but specific components then one general with increased complexity. Less is often more. |

### Configuration
Configuration values can also be automatically injected with arguments or environment variables at start time. The value from ```USER``` will be used in this example. If the argument ```--USER madpax``` is not set and the environment variable is not defined, it is possible to specify the reaction whether the execution should stop with a panic or continue with a warning.
```go
package main

import (
	"github.com/boot-go/boot"
	"log"
)

// hello is still a simple component.
type hello struct{
	Out string `boot:"config,key:USER,default:madjax"` // get the value from the argument list or environment variable. If no value could be determined, then use the default value `madjax`.
}

// init() registers a factory method, which creates a hello component and returns a reference to it.
func init() {
	boot.Register(func() boot.Component {
		return &hello{}
	})
}

// Init is the initializer of the component.
func (c *hello) Init() error {
	log.Printf("boot-go says > 'Hello %s'\n", c.Out)
	return nil
}

// Start the example and exit after the component was completed
func main() {
	boot.Go()
}

```


### boot stack
**boot-go** was primarily designed to build opinionated frameworks and bundle them as a stack. So every developer or company can choose to use the [default stack](https://github.com/boot-go/stack), a shared stack or rather create a new one. Stacks should be build with one specific purpose in mind for building a **microservice**, **ui application**, **web application**, **data analytics application** and so on. As an example, a **web application boot stack** could contain a http server component, a sql database component, a logging and a web application framework.


### Examples
More examples can be found in the [tutorial repository](https://github.com/boot-go/tutorial).
