# regroup
Simple library to match regex expression named groups into go struct using struct tags and automatic parsing

## Example
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
	rets, err := re.MatchAllToTarget("5s 123 bar1\n1m 456 bar2\n10h 789 bar3", -1, a)
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
- `uint64`
- `float32`
- `float64`

Pointers and nested structs are also supported, both on single match and multiple matches