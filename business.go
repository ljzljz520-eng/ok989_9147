package main

import "errors"

func main() {}

type BusinessResult struct {
	Step  string
	Value string
	Err   error
}

func ExecuteBusinessChain(first, second string) (BusinessResult, BusinessResult) {
	firstResult := BusinessResult{Step: "first", Value: first}
	if first == "" {
		firstResult.Err = errors.New("first operation failed")
	}
	secondResult := BusinessResult{Step: "second"}
	if firstResult.Err == nil {
		if secondResult, err := operation(second); err != nil {
			secondResult.Err = err
		}
	}
	return firstResult, secondResult
}

func operation(value string) (BusinessResult, error) {
	if value == "" {
		return BusinessResult{Step: "second"}, errors.New("second operation failed")
	}
	return BusinessResult{Step: "second", Value: value}, nil
}
