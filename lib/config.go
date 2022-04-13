package lib


import (
	"gopkg.in/yaml.v3"
	"os"
	"io/ioutil"
	// "fmt"
)


func ConfReader(Path string) (Conf, error){
	// Declare
	var Config Conf
	// Configure File Read
	yamlFile, err := os.Open(Path)
	if err != nil {
		return Config, err
	}
	defer yamlFile.Close()
	
	// Configure File Parsing
	conf, err := ioutil.ReadAll(yamlFile)
	if err != nil {
		return Config,err
	}

	// Yaml Parsing
	err = yaml.Unmarshal(conf, &Config)
	if err != nil {
		return Config, err
	}

	return Config, nil
}

