package followdata

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/cecobask/instagram-insights/pkg/filesystem"
	"github.com/cecobask/instagram-insights/pkg/instagram"
	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

type Interface interface {
	Followers(opts *instagram.Options) (*string, error)
	Following(opts *instagram.Options) (*string, error)
	Unfollowers(opts *instagram.Options) (*string, error)
}

type handler struct {
	fileSystem filesystem.Fs
	followData *followData
}

func NewHandler() Interface {
	return &handler{
		fileSystem: filesystem.NewFs(),
		followData: newFollowData(),
	}
}

func (h *handler) Followers(opts *instagram.Options) (*string, error) {
	files, err := h.fileSystem.FindFiles(instagram.PathFollowers)
	if err != nil {
		return nil, err
	}
	for i := range files {
		data, err := h.fileSystem.ReadFile(files[i])
		if err != nil {
			return nil, err
		}
		if err = h.followData.hydrateFollowers(data); err != nil {
			return nil, err
		}
	}
	h.followData.Followers.Sort(opts.SortBy, opts.Order)
	h.followData.Followers.Limit(opts.Limit)
	return h.followData.Followers.output(opts.Output)
}

func (h *handler) Following(opts *instagram.Options) (*string, error) {
	data, err := h.fileSystem.ReadFile(instagram.PathFollowing)
	if err != nil {
		return nil, err
	}
	if err = h.followData.hydrateFollowing(data); err != nil {
		return nil, err
	}
	h.followData.Following.Sort(opts.SortBy, opts.Order)
	h.followData.Following.Limit(opts.Limit)
	return h.followData.Following.output(opts.Output)
}

func (h *handler) Unfollowers(opts *instagram.Options) (*string, error) {
	emptyOptions := instagram.NewEmptyOptions()
	if _, err := h.Followers(emptyOptions); err != nil {
		return nil, err
	}
	if _, err := h.Following(emptyOptions); err != nil {
		return nil, err
	}
	h.followData.hydrateUnfollowers()
	h.followData.Unfollowers.Sort(opts.SortBy, opts.Order)
	h.followData.Unfollowers.Limit(opts.Limit)
	return h.followData.Unfollowers.output(opts.Output)
}

type followData struct {
	Following   *userList
	Followers   *userList
	Unfollowers *userList
}

func newFollowData() *followData {
	return &followData{
		Following:   newUserList(true),
		Followers:   newUserList(true),
		Unfollowers: newUserList(false),
	}
}

type followers []struct {
	StringListData []struct {
		Href      string `json:"href"`
		Value     string `json:"value"`
		Timestamp int    `json:"timestamp"`
	} `json:"string_list_data"`
}

func (fd *followData) hydrateFollowers(data []byte) error {
	var fs followers
	if err := json.Unmarshal(data, &fs); err != nil {
		return err
	}
	for _, f := range fs {
		fd.Followers.Append(user{
			ProfileUrl: f.StringListData[0].Href,
			Username:   f.StringListData[0].Value,
			Timestamp: &timestamp{
				Time: time.Unix(int64(f.StringListData[0].Timestamp), 0),
			},
		})
	}
	return nil
}

type following struct {
	RelationshipsFollowing []struct {
		Title          string `json:"title"`
		StringListData []struct {
			Href      string `json:"href"`
			Timestamp int    `json:"timestamp"`
		} `json:"string_list_data"`
	} `json:"relationships_following"`
}

func (fd *followData) hydrateFollowing(data []byte) error {
	var f following
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	for _, rf := range f.RelationshipsFollowing {
		fd.Following.Append(user{
			ProfileUrl: rf.StringListData[0].Href,
			Username:   rf.Title,
			Timestamp: &timestamp{
				Time: time.Unix(int64(rf.StringListData[0].Timestamp), 0),
			},
		})
	}
	return nil
}

func (fd *followData) hydrateUnfollowers() {
	for i := range fd.Following.users {
		current := fd.Following.users[i]
		index := slices.IndexFunc(fd.Followers.users, func(u user) bool {
			return u.Username == current.Username
		})
		if index == -1 {
			fd.Unfollowers.Append(current)
		}
	}
}

type timestamp struct {
	time.Time
}

func (t *timestamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.String())), nil
}

func (t *timestamp) UnmarshalJSON(b []byte) error {
	var unixTimestamp int64
	if err := json.Unmarshal(b, &unixTimestamp); err != nil {
		return err
	}
	t.Time = time.Unix(unixTimestamp, 0)
	return nil
}

func (t *timestamp) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *timestamp) String() string {
	return t.Format(time.RFC3339)
}

type user struct {
	ProfileUrl string     `json:"profileUrl" yaml:"profileUrl"`
	Username   string     `json:"username" yaml:"username"`
	Timestamp  *timestamp `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
}

type userList struct {
	users         []user
	showTimestamp bool
}

func newUserList(showTimestamp bool) *userList {
	return &userList{
		users:         make([]user, 0),
		showTimestamp: showTimestamp,
	}
}

func (ul *userList) output(format string) (*string, error) {
	switch format {
	case instagram.OutputJson:
		return ul.outputJson()
	case instagram.OutputNone:
		return ul.outputNone()
	case instagram.OutputTable:
		return ul.outputTable()
	case instagram.OutputYaml:
		return ul.outputYaml()
	default:
		return nil, fmt.Errorf("invalid output format: %s", format)
	}
}

func (ul *userList) outputNone() (*string, error) {
	output := ""
	return &output, nil
}

func (ul *userList) outputJson() (*string, error) {
	if !ul.showTimestamp {
		for i := range ul.users {
			ul.users[i].Timestamp = nil
		}
	}
	data, err := json.MarshalIndent(ul.users, "", "  ")
	if err != nil {
		return nil, err
	}
	output := string(data)
	return &output, nil
}

func (ul *userList) outputTable() (*string, error) {
	var rows []table.Row
	for i := range ul.users {
		current := ul.users[i]
		row := table.Row{
			current.Username,
			current.ProfileUrl,
		}
		if ul.showTimestamp {
			row = append(row, current.Timestamp)
		}
		rows = append(rows, row)
	}
	header := table.Row{
		instagram.TableHeaderUsername,
		instagram.TableHeaderProfileUrl,
	}
	if ul.showTimestamp {
		header = append(header, instagram.TableHeaderTimestamp)
	}
	usersTable := table.NewWriter()
	usersTable.SetAutoIndex(true)
	usersTable.SetStyle(table.StyleBold)
	usersTable.AppendHeader(header)
	usersTable.AppendRows(rows)
	output := usersTable.Render()
	return &output, nil
}

func (ul *userList) outputYaml() (*string, error) {
	if !ul.showTimestamp {
		for i := range ul.users {
			ul.users[i].Timestamp = nil
		}
	}
	data, err := yaml.Marshal(ul.users)
	if err != nil {
		return nil, err
	}
	output := string(data)
	return &output, nil
}

func (ul *userList) Sort(field string, order string) {
	sort.Slice(ul.users, func(a, b int) bool {
		userOne := ul.users[a]
		userTwo := ul.users[b]
		switch field {
		case instagram.FieldTimestamp:
			return userOne.Timestamp.Before(userTwo.Timestamp.Time)
		case instagram.FieldUsername:
			return userOne.Username < userTwo.Username
		default:
			return userOne.Timestamp.Before(userTwo.Timestamp.Time)
		}
	})
	if order == instagram.OrderDesc {
		slices.Reverse(ul.users)
	}
}

func (ul *userList) Limit(limit int) {
	if limit > 0 && limit < len(ul.users) {
		ul.users = ul.users[:limit]
	}
}

func (ul *userList) Append(u user) {
	ul.users = append(ul.users, u)
}
