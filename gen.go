package gopter

import (
	"fmt"
	"reflect"
)

// Gen generator of arbitrary values.
// Usually properties are checked by verifing a condition holds true for arbitrary input parameters
type Gen func(*GenParameters) *GenResult

// Sample generate a sample value.
// Depending on the state of the RNG the generate might fail to provide a sample
func (g Gen) Sample() (interface{}, bool) {
	return g(DefaultGenParameters()).Retrieve()
}

// WithLabel adds a label to a generated value.
// Labels are usually used for reporting for the arguments of a property check.
func (g Gen) WithLabel(label string) Gen {
	return func(genParams *GenParameters) *GenResult {
		result := g(genParams)
		result.Labels = append(result.Labels, label)
		return result
	}
}

// SuchThat creates a derived generator by adding a sieve.
// f: has to be a function with one parameter (matching the generated value) returning a bool.
// All generated values are expected to satisfy
//  f(value) == true.
// Use this care, if the sieve to to fine the generator will have many misses which results
// in an undecided property.
func (g Gen) SuchThat(f interface{}) Gen {
	checkVal := reflect.ValueOf(f)
	checkType := checkVal.Type()

	if checkVal.Kind() != reflect.Func {
		panic(fmt.Sprintf("Param of SuchThat has to be a func, but is %v", checkType.Kind()))
	}
	if checkType.NumIn() != 1 {
		panic(fmt.Sprintf("Param of SuchThat has to be a func with one param, but is %v", checkType.NumIn()))
	} else {
		genResultType := g(DefaultGenParameters()).ResultType
		if !genResultType.AssignableTo(checkType.In(0)) {
			panic(fmt.Sprintf("Param of SuchThat has to be a func with one param assignable to %v, but is %v", genResultType, checkType.In(0)))
		}
	}
	if checkType.NumOut() != 1 {
		panic(fmt.Sprintf("Param of SuchThat has to be a func with one return value, but is %v", checkType.NumOut()))
	} else if checkType.Out(0).Kind() != reflect.Bool {
		panic(fmt.Sprintf("Param of SuchThat has to be a func with one return value of bool, but is %v", checkType.Out(0).Kind()))
	}
	sieve := func(v interface{}) bool {
		return checkVal.Call([]reflect.Value{reflect.ValueOf(v)})[0].Bool()
	}

	return func(genParams *GenParameters) *GenResult {
		result := g(genParams)
		prevSieve := result.Sieve
		if prevSieve == nil {
			result.Sieve = sieve
		} else {
			result.Sieve = func(value interface{}) bool {
				return prevSieve(value) && sieve(value)
			}
		}
		return result
	}
}

// WithShrinker creates a derived generator with a specific shrinker
func (g Gen) WithShrinker(shrinker Shrinker) Gen {
	return func(genParams *GenParameters) *GenResult {
		result := g(genParams)
		if shrinker == nil {
			result.Shrinker = NoShrinker
		} else {
			result.Shrinker = shrinker
		}
		return result
	}
}

// Map creates a derived generators by mapping all generatored values with a given function.
// f: has to be a function with one parameter (matching the generated value) and a single return.
// Note: The derived generator will not have a sieve or shrinker.
func (g Gen) Map(f interface{}) Gen {
	mapperVal := reflect.ValueOf(f)
	mapperType := mapperVal.Type()

	if mapperVal.Kind() != reflect.Func {
		panic(fmt.Sprintf("Param of Map has to be a func, but is %v", mapperType.Kind()))
	}
	if mapperType.NumIn() != 1 {
		panic(fmt.Sprintf("Param of Map has to be a func with one param, but is %v", mapperType.NumIn()))
	} else {
		genResultType := g(DefaultGenParameters()).ResultType
		if !genResultType.AssignableTo(mapperType.In(0)) {
			panic(fmt.Sprintf("Param of Map has to be a func with one param assignable to %v, but is %v", genResultType, mapperType.In(0)))
		}
	}
	if mapperType.NumOut() != 1 {
		panic(fmt.Sprintf("Param of Map has to be a func with one return value, but is %v", mapperType.NumOut()))
	}

	return func(genParams *GenParameters) *GenResult {
		result := g(genParams)
		value, ok := result.RetrieveAsValue()
		if ok {
			mapped := mapperVal.Call([]reflect.Value{value})[0]
			return &GenResult{
				Shrinker:   NoShrinker,
				result:     mapped.Interface(),
				Labels:     result.Labels,
				ResultType: mapperType.Out(0),
			}
		}
		return &GenResult{
			Shrinker:   NoShrinker,
			result:     nil,
			Labels:     result.Labels,
			ResultType: mapperType.Out(0),
		}
	}
}

// FlatMap creates a derived generator by passing a generated value to a function which itself
// creates a generator.
func (g Gen) FlatMap(f func(interface{}) Gen, resultType reflect.Type) Gen {
	return func(genParams *GenParameters) *GenResult {
		result := g(genParams)
		value, ok := result.Retrieve()
		if ok {
			return f(value)(genParams)
		}
		return &GenResult{
			Shrinker:   NoShrinker,
			result:     nil,
			Labels:     result.Labels,
			ResultType: resultType,
		}
	}
}

// CombineGens creates a generators from a list of generators.
// The result type will be a []interface{} containing the generated values of each generators in
// the list.
// Note: The combined generator will not have a sieve or shrinker.
func CombineGens(gens ...Gen) Gen {
	return func(genParams *GenParameters) *GenResult {
		labels := []string{}
		values := make([]interface{}, len(gens))
		shrinkers := make([]Shrinker, len(gens))
		sieves := make([]func(v interface{}) bool, len(gens))

		var ok bool
		for i, gen := range gens {
			result := gen(genParams)
			labels = append(labels, result.Labels...)
			shrinkers[i] = result.Shrinker
			sieves[i] = result.Sieve
			values[i], ok = result.Retrieve()
			if !ok {
				return &GenResult{
					Shrinker:   NoShrinker,
					result:     nil,
					Labels:     result.Labels,
					ResultType: reflect.TypeOf(values),
				}
			}
		}
		return &GenResult{
			Shrinker:   CombineShrinker(shrinkers...),
			result:     values,
			Labels:     labels,
			ResultType: reflect.TypeOf(values),
			Sieve: func(v interface{}) bool {
				values := v.([]interface{})
				for i, value := range values {
					if sieves[i] != nil && !sieves[i](value) {
						return false
					}
				}
				return true
			},
		}
	}
}
