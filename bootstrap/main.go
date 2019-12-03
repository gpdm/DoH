package main

// This file is used purely for compiling project dependencies in the absence of
// the project source. this allows us to cache those deps in a separate docker
// layer and avoids having to do a full recompile when building new images after
// every source change.

// Add here any deps with large build times.
import (
	_ "github.com/sirupsen/logrus"
	_ "github.com/spf13/viper"
)

func main() {

}
