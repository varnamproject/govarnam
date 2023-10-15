package govarnam

type DictionaryConfig struct {
	path  string
	write bool // Use for storing user learnt words
}

func (varnam *Varnam) setDefaultDictionariesConfig() {
	systemDictPath, err := findSystemDictionaryPath(varnam.SchemeDetails.LangCode)

	if err == nil {
		varnam.DictsConfig = append(varnam.DictsConfig, DictionaryConfig{systemDictPath, false})
	}

	userDictPath := findUserDictionaryPath(varnam.SchemeDetails.LangCode)

	varnam.DictsConfig = append(varnam.DictsConfig, DictionaryConfig{userDictPath, true})
}

func (varnam *Varnam) InitDicts() error {
	for _, dictConfig := range varnam.DictsConfig {
		if dictConfig.write {
			err := varnam.InitDict(dictConfig.path)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
