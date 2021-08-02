package lib

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Abstract descriptor for determination type of content in files.
type descriptor struct {
	Type string `json:"-" yaml:"type"`
	List []byte `json:"list" yaml:"list"`
}

// ApplyFile is function, which read file and check updates for object
// from file.
func (m *MoiraAPI) ApplyFile(file string) {
	if file == "" {
		log.Fatal("File must be not empty, apply cancelled!")
	}
	size := checkFile(file)
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()
	var object = make([]byte, 0)
	data := make([]byte, size)
	for {
		n, err := f.Read(data)
		if err == io.EOF {
			break
		}
		object = append(object, data[:n]...)
	}
	var result descriptor
	err = yaml.Unmarshal(object, &result)
	if err != nil {
		return
	}
	switch result.Type {
	case "trigger":
		if err := m.setTrigger(object); err != nil {
			fmt.Print(err.Error())
		}
	case "user":
		if err := m.setUserSettings(object); err != nil {
			fmt.Print(err.Error())
		}
	case "tag":
		fmt.Printf("Tags autocreating when trigger create\n")
	}
}

// DumpToDir getting all information about triggers, tags and users and write it
// into dir.
func (m *MoiraAPI) DumpToDir(dir string) {
	var (
		tags          *tags
		triggers      []trigger
		usersSettings []userSettings
		err           error
	)
	if dir == "" {
		log.Fatal("Directory must be not empty, dump cancelled!")
	}

	// fmt.Printf("Dumping tags...")
	if tags, err = m.getAllTags(); err != nil {
		log.Fatal(err.Error())
	}

	// fmt.Printf("Dumping triggers...")
	if triggers, err = m.getAllTriggers(); err != nil {
		log.Fatal(err.Error())
	}

	// fmt.Printf("Dumping users settings...")
	if usersSettings, err = m.getAllUsersSettings(); err != nil {
		log.Fatal(err.Error())
	}

	var files = []string{"tags", "triggers", "users"}
	var totals = map[string]int{"tags": 0, "triggers": 0, "users": 0}
	for _, file := range files {
		checkDir(fmt.Sprintf("%s/%s", dir, file))
		switch file {
		case "tags":
			path := fmt.Sprintf("%s/%s/%s.yml", dir, file, file)
			fmt.Printf("Saving tags to %s\n", path)
			writeData(tags, path)
			totals[file] = len(tags.List)
		case "triggers":
			for _, trigger := range triggers {
				name := trigger.Name
				name = strings.ReplaceAll(name, " ", "_")
				path := fmt.Sprintf("%s/%s/%s.yml", dir, file, name)
				// ID not need to save.
				trigger.ID = ""
				fmt.Printf("Saving trigger '%s' to %s\n", trigger.Name, path)
				writeData(trigger, path)
				totals[file]++
			}
		case "users":
			for _, user := range usersSettings {
				path := fmt.Sprintf("%s/%s/%s.yml", dir, file, user.Login)
				fmt.Printf("Saving user '%s' to %s\n", user.Login, path)
				writeData(user, path)
				totals[file]++
			}
		default:
			log.Fatal("Unknown file")
		}
	}

	fmt.Printf("Total saved:\n")
	for index, value := range totals {
		fmt.Printf("%s : %d\n", index, value)
	}
}

// checkDir checked what dir is directory and recreate it.
// Fatal if dir not a directory.
func checkDir(dir string) {
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0700)
	} else {
		if stat.IsDir() {
			os.RemoveAll(dir)
			_ = os.MkdirAll(dir, 0700)
		} else {
			log.Fatalf("%s is not a directory!", dir)
		}
	}
}

// checkDir checked what 'file' is exist file.
func checkFile(file string) int64 {
	stat, err := os.Stat(file)
	if os.IsNotExist(err) {
		log.Fatalf("%s does not exist!", file)
	} else if stat.IsDir() {
		log.Fatalf("%s is a directory!", file)
	}
	return stat.Size()
}

// writeData writed 'data' in YAML format to 'file'.
func writeData(data interface{}, file string) {
	dataM, _ := yaml.Marshal(&data)
	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err.Error())
	}
	_, _ = f.Write(dataM)
	f.Close()
}
