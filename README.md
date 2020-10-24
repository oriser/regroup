# regroup
Simple library to match regex expression named groups into go struct using struct tags and automatic parsing

![](https://github.com/oriser/regroup/workflows/reviewdog/badge.svg)
![](https://github.com/oriser/regroup/workflows/Go/badge.svg)
[![codecov](https://codecov.io/gh/oriser/regroup/branch/master/graph/badge.svg)](https://codecov.io/gh/oriser/regroup)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/oriser/regroup)](https://pkg.go.dev/github.com/oriser/regroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/oriser/regroup?a=b)](https://goreportcard.com/report/github.com/oriser/regroup)
[![codeclimate](https://api.codeclimate.com/v1/badges/169ebfa87cb6af0c6db6/maintainability)](https://goreportcard.com/report/github.com/oriser/regroup)

### Installing
`go get github.com/oriser/regroup`


## Example
#### Named groups map
```go
package main

import (
	"fmt"
	"github.com/oriser/regroup"
)

var re = regroup.MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<foo>.*)`)

func main() {
	mathces, err := re.Groups("5s 123 bar")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", mathces)
}
```
Will output:
`map[duration:5s foo:bar num:123]`

#### Single match
```go
package main

import (
	"fmt"
	"github.com/oriser/regroup"
	"time"
)

var re = regroup.MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<foo>.*)`)

type B struct {
	Str string `regroup:"foo"`
}

type A struct {
	Number        int           `regroup:"num"`
	Dur           time.Duration `regroup:"duration"`
	AnotherStruct B
}

func main() {
	a := &A{}
	if err := re.MatchToTarget("5s 123 bar", a); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", a)
}

```
Will output:
`&{Number:123 Dur:5s AnotherStruct:{Str:bar}}`

#### Multiple matches
You can also get all matches parsed as given target struct. The return value in this
case will be an array of interfaces, you should cast it to the target type in order to access its fields.
```go
package main

import (
	"fmt"
	"github.com/oriser/regroup"
	"time"
)

var re = regroup.MustCompile(`\s*(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<foo>.*)`)

type B struct {
	Str string `regroup:"foo"`
}

type A struct {
	Number        int           `regroup:"num"`
	Dur           time.Duration `regroup:"duration"`
	AnotherStruct B
}

func main() {
	a := &A{}
	s := `5s 123 bar1
		  1m 456 bar2
		  10h 789 bar3`
	rets, err := re.MatchAllToTarget(s, -1, a)
	if err != nil {
		panic(err)
	}
	for _, elem := range rets {
		fmt.Printf("%+v\n", elem.(*A))
	}
}

```
Will output:
```
&{Number:123 Dur:5s AnotherStruct:{Str:bar1}}
&{Number:456 Dur:1m0s AnotherStruct:{Str:bar2}}
&{Number:789 Dur:10h0m0s AnotherStruct:{Str:bar3}}
```

#### Required groups
You can specify that a specific group is required, means that it can't be empty.

If a required group is empty, an error (`*regroup.RequiredGroupIsEmpty`) will be returned .
```go
package main

import (
	"fmt"
	"github.com/oriser/regroup"
	"time"
)

var re = regroup.MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<foo>.*)`)

type B struct {
	Str string `regroup:"foo,required"`
}

type A struct {
	Number        int           `regroup:"num"`
	Dur           time.Duration `regroup:"duration"`
	AnotherStruct B
}

func main() {
	a := &A{}
	if err := re.MatchToTarget("5s 123 ", a); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", a)
}
```
Will return an error: `required regroup "foo" is empty for field "Str"`

## Supported struct field types
- `time.Duration`
- `bool`
- `string`
- `int`
- `int8`
- `int16`
- `int32`
- `int64`
- `uint`
- `uint8`
- `uint16`
- `uint32`
- `uint64`
- `float32`
- `float64`

Pointers and nested structs are also supported, both on single match and multiple matches
