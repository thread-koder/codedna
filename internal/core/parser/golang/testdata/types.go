package types

type User struct {
	ID       int
	Name     string
	Email    string
	Active   bool
	Metadata map[string]interface{}
	Tags     []string
	Friends  []*User
	Settings *UserSettings
	Messages chan string
}

type UserSettings struct {
	Theme        string
	Preferences  map[string]bool
	Permissions  []string
	Notification struct {
		Email    bool
		Push     bool
		Interval int
	}
}

func (u *User) UpdateSettings(settings *UserSettings) {
	u.Settings = settings
}

func (u *User) SendMessage(msg string) {
	u.Messages <- msg
}
