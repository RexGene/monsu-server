package configmanager

import (
	"github.com/RexGene/csvparser"
)

var instance *ConfigManager

type ConfigManager struct {
	dataMap map[string]csvparser.Result
}

func GetInstance() *ConfigManager {
	if instance == nil {
		instance = &ConfigManager{
			dataMap: make(map[string]csvparser.Result),
		}
	}

	return instance
}

func (this *ConfigManager) Clear() {
	this.dataMap = make(map[string]csvparser.Result)
}

func (this *ConfigManager) GetConfig(configName string) (result csvparser.Result, err error) {
	result = this.dataMap[configName]
	if result == nil {
		result, err = csvparser.Parse(configName)
		if err != nil {
			return nil, err
		}

		this.dataMap[configName] = result
	}

	return result, nil
}
