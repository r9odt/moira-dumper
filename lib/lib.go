package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

// MoiraAPI is api class.
type MoiraAPI struct {
	API string
}

// triggers describe api response for /api/trigger.
type triggers struct {
	List []trigger `json:"list" yaml:"-"`
}

type sched struct {
	Days        []day `json:"days" yaml:"days,omitempty"`
	TzOffset    int64 `json:"tzOffset" yaml:"tzOffset,omitempty"`
	StartOffset int64 `json:"startOffset" yaml:"startOffset,omitempty"`
	EndOffset   int64 `json:"endOffset" yaml:"endOffset,omitempty"`
}

type day struct {
	Enabled bool   `json:"enabled" yaml:"enabled,omitempty"`
	Name    string `json:"name" yaml:"name,omitempty"`
}

func checkEmptySched(schedule *sched) {
	if len((*schedule).Days) == 0 {
		*schedule = getDefaultSchedule()
	}
}

func checkDefaultSchedAndSetEmpty(schedule *sched) bool {
	def := getDefaultSchedule()
	s := *schedule
	daysEqual := true
	for i := range s.Days {
		if s.Days[i] != def.Days[i] {
			daysEqual = false
		}
	}
	if s.TzOffset == def.TzOffset &&
		s.StartOffset == def.StartOffset &&
		s.EndOffset == def.EndOffset &&
		daysEqual {
		*schedule = getEmptySchedule()
		return false
	}
	return true
}

func getEmptySchedule() sched {
	return sched{
		Days:        []day{},
		TzOffset:    0,
		StartOffset: 0,
		EndOffset:   0,
	}
}

func getDefaultSchedule() sched {
	return sched{
		Days: []day{
			{true, "Mon"},
			{true, "Tue"},
			{true, "Wed"},
			{true, "Thu"},
			{true, "Fri"},
			{true, "Sat"},
			{true, "Sun"}},
		TzOffset:    -420,
		StartOffset: 0,
		EndOffset:   1439,
	}
}

type trigger struct {
	Type        string   `json:"-" yaml:"type"`
	ID          string   `json:"id" yaml:"id,omitempty"`
	Name        string   `json:"name" yaml:"name,omitempty"`
	Desc        string   `json:"desc" yaml:"desc,omitempty"`
	Targets     []string `json:"targets" yaml:"targets,omitempty"`
	TriggerType string   `json:"trigger_type" yaml:"trigger_type,omitempty"`
	WarnValue   float64  `json:"warn_value,omitempty" yaml:"warn_value,omitempty"`
	ErrorValue  float64  `json:"error_value,omitempty" yaml:"error_value,omitempty"`
	Expression  string   `json:"expression" yaml:"expression,omitempty"`
	Tags        []string `json:"tags" yaml:"tags,omitempty"`
	Sched       sched    `json:"sched" yaml:"sched,omitempty"`
	TTLState    string   `json:"ttl_state" yaml:"ttl_state,omitempty"`
	TTL         int64    `json:"ttl" yaml:"ttl,omitempty"`
	IsRemote    bool     `json:"is_remote" yaml:"is_remote,omitempty"`
}

func (m *MoiraAPI) getAllTriggers() ([]trigger, error) {
	var responseMapper triggers
	var url = fmt.Sprintf("%s/trigger", m.API)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &responseMapper)
	for index := range responseMapper.List {
		responseMapper.List[index].Type = "trigger"
		checkDefaultSchedAndSetEmpty(&responseMapper.List[index].Sched)
	}
	return responseMapper.List, err
}

func (m *MoiraAPI) setTrigger(object []byte) error {
	var requestMapper trigger
	var url = fmt.Sprintf("%s/trigger", m.API)
	var existingTriggers []trigger
	var err error
	if existingTriggers, err = m.getAllTriggers(); err != nil {
		return err
	}

	var triggerMapper = make(map[string]int)
	if err = yaml.Unmarshal(object, &requestMapper); err != nil {
		return err
	}

	for index, trigger := range existingTriggers {
		triggerMapper[trigger.Name] = index
	}
	checkEmptySched(&requestMapper.Sched)

	needRequest := false
	needUpdate := false
	if _, ok := triggerMapper[requestMapper.Name]; !ok {
		needRequest = true
		fmt.Printf("Create trigger '%s'\n", requestMapper.Name)
	} else {
		fmt.Printf("Trigger '%s' already exist, id: %s\n", requestMapper.Name,
			existingTriggers[triggerMapper[requestMapper.Name]].ID)
		if checkTriggerFieldsUpdate(requestMapper,
			existingTriggers[triggerMapper[requestMapper.Name]]) {
			fmt.Printf("Trigger '%s' has updated fields\n", requestMapper.Name)
			needRequest = true
			needUpdate = true
		}
	}
	if needRequest {
		if needUpdate {
			requestMapper.ID = existingTriggers[triggerMapper[requestMapper.Name]].ID
			url = fmt.Sprintf("%s/%s", url, requestMapper.ID)
		}
		requestJSON, err := json.Marshal(requestMapper)
		if err != nil {
			return nil
		}
		client := http.Client{
			Timeout: 5 * time.Second,
		}

		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestJSON))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		response, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("request complete with code %d. %s", resp.StatusCode, response)
		}
	}
	return nil
}

func checkTriggerFieldsUpdate(request, existing trigger) bool {
	if request.Desc != existing.Desc ||
		request.TriggerType != existing.TriggerType ||
		request.TTLState != existing.TTLState ||
		request.TTL != existing.TTL ||
		request.IsRemote != existing.IsRemote {
		// ||
		// request.Sched.TzOffset != existing.Sched.TzOffset ||
		// request.Sched.StartOffset != existing.Sched.StartOffset ||
		// request.Sched.EndOffset != existing.Sched.EndOffset
		return true
	}

	// for i, _ := range request.Sched.Days {
	// 	if request.Sched.Days[i] != existing.Sched.Days[i] {
	// 		return true
	// 	}
	// }

	for i := range request.Targets {
		if request.Targets[i] != existing.Targets[i] {
			return true
		}
	}

	switch existing.TriggerType {
	case "falling", "rising":
		if request.WarnValue != existing.WarnValue ||
			request.ErrorValue != existing.ErrorValue {
			return true
		}
	case "expression":
		if request.Expression != existing.Expression {
			return true
		}
	default:
		fmt.Printf("Trigger type '%s' not supported!\n", existing.TriggerType)
		return false
	}
	// if request. != existing.TriggerType {
	// 	return true
	// }
	return false
}

// tags describe api response for /api/tag.
type tags struct {
	Type string   `json:"-" yaml:"type"`
	List []string `json:"list" yaml:"list"`
}

func (m *MoiraAPI) getAllTags() (*tags, error) {
	var responseMapper tags
	var url = fmt.Sprintf("%s/tag", m.API)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &responseMapper)
	responseMapper.Type = "tag"
	return &responseMapper, err
}

// contacts describe api response for /api/contact.
type contacts struct {
	List []contact `json:"list" yaml:"-"`
}

type contact struct {
	ID    string `json:"id,omitempty" yaml:"-"`
	Type  string `json:"type" yaml:"type,omitempty"`
	User  string `json:"user" yaml:"-"`
	Value string `json:"value" yaml:"value,omitempty"`
}

func (m *MoiraAPI) getAllContacts() ([]contact, error) {
	var responseMapper contacts
	var url = fmt.Sprintf("%s/contact", m.API)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &responseMapper)
	return responseMapper.List, err
}

type userSettings struct {
	Type  string `json:"-" yaml:"type"`
	Login string `json:"login" yaml:"login,omitempty"`
	// Contacts      []contact      `json:"contacts" yaml:"contacts,omitempty"`
	Subscriptions []subscription `json:"subscriptions" yaml:"subscriptions" `
}

// subscriptions describe api response for /api/subscription.
// type subscriptions struct {
// 	List []subscription `json:"list,omitempty"`
// }

type plotting struct {
	Enabled bool   `json:"enabled" yaml:"enabled,omitempty"`
	Theme   string `json:"theme" yaml:"theme,omitempty"`
}

type subscription struct {
	// ID             string    `json:"id" yaml:"id,omitempty"`
	User           string    `json:"user" yaml:"-"`
	Contacts       []string  `json:"contacts" yaml:"-"`
	ContactsByID   []contact `json:"-" yaml:"contacts,omitempty"`
	Tags           []string  `json:"tags" yaml:"tags,omitempty"`
	Sched          sched     `json:"sched" yaml:"sched,omitempty"`
	Plotting       plotting  `json:"plotting" yaml:"plotting,omitempty"`
	Enabled        bool      `json:"enabled" yaml:"enabled,omitempty"`
	AnyTags        bool      `json:"any_tags" yaml:"any_tags,omitempty"`
	IgnoreWarnings bool      `json:"ignore_warnings" yaml:"ignore_warnings,omitempty"`
	Throttling     bool      `json:"throttling" yaml:"throttling,omitempty"`
}

func (m *MoiraAPI) getAllUsersSettings() ([]userSettings, error) {
	// var response userSettings
	var (
		contacts       []contact
		err            error
		usersMapper    = make(map[string]bool)
		contactsMapper = make(map[string]contact)
		responseMapper = make([]userSettings, 0)
	)
	if contacts, err = m.getAllContacts(); err != nil {
		return nil, err

	}
	for _, contact := range contacts {
		contactsMapper[contact.ID] = contact
		usersMapper[contact.User] = true
	}

	var url = fmt.Sprintf("%s/user/settings", m.API)

	// Parse contacts and foreach with X-WebAuth-User.
	for user := range usersMapper {
		response := userSettings{}
		client := http.Client{
			Timeout: 5 * time.Second,
		}

		// resp, err := client.Get(url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-WebAuth-User", user)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		err = json.Unmarshal(body, &response)
		if err != nil {
			continue
		}
		for index, subscription := range response.Subscriptions {
			checkDefaultSchedAndSetEmpty(&response.Subscriptions[index].Sched)
			for _, id := range subscription.Contacts {
				contactElement := contactsMapper[id]
				response.Subscriptions[index].ContactsByID =
					append(response.Subscriptions[index].ContactsByID,
						contactElement)
			}
		}
		response.Type = "user"
		responseMapper = append(responseMapper, response)
	}

	return responseMapper, err
}

func (m *MoiraAPI) setUserSettings(object []byte) error {
	// TODO: Applying subscriptions.
	var (
		requestMapper  userSettings
		contactURL     = fmt.Sprintf("%s/contact", m.API)
		contacts       []contact
		usersMapper    = make(map[string]bool)
		contactsMapper = make(map[string]contact)
		err            error
	)

	if contacts, err = m.getAllContacts(); err != nil {
		return err
	}

	for _, contact := range contacts {
		contactsMapper[contact.ID] = contact
		usersMapper[contact.User] = true
	}

	if err = yaml.Unmarshal(object, &requestMapper); err != nil {
		return err
	}

	login := requestMapper.Login
	fmt.Printf("Set login as '%s'\n", login)

	for _, subscription := range requestMapper.Subscriptions {
		for _, currentContact := range subscription.ContactsByID {
			contactFound := false
			for id, c := range contactsMapper {
				if currentContact.User == login {
					if currentContact.Type == c.Type && currentContact.Value == c.Value {
						subscription.Contacts = append(subscription.Contacts, id)
						contactFound = true
						break
					}
				}
			}
			if !contactFound {
				c := contact{
					ID:    "",
					User:  login,
					Type:  currentContact.Type,
					Value: currentContact.Value,
				}

				requestJSON, err := json.Marshal(c)
				if err != nil {
					return nil
				}

				client := http.Client{
					Timeout: 5 * time.Second,
				}

				req, err := http.NewRequest(http.MethodPut, contactURL, bytes.NewBuffer(requestJSON))
				if err != nil {
					return err
				}

				fmt.Printf("Request %s\n", requestJSON)
				req.Header.Set("Content-Type", "application/json; charset=utf-8")
				req.Header.Set("X-WebAuth-User", login)
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				response, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				if resp.StatusCode == 200 {
					err = json.Unmarshal(response, &c)
					if err != nil {
						return err
					}
					contactsMapper[c.ID] = c

					fmt.Printf("Contact '%#v' for user '%s' was create\n", c, login)
				}
			}
		}
	}

	return nil
}
