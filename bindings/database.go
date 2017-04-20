package bindings

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/clixxa/dsp/services"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"log"
	"strconv"
	"strings"
	"time"
)

func tojson(i interface{}) string {
	s, _ := json.Marshal(i)
	return string(s)
}

func wide(depth int) string {
	return strings.Repeat("\t", depth)
}

const sqlUserIPs = `SELECT ip FROM ip_histories WHERE user_id = ?`
const sqlUser = `SELECT setting_id, value FROM user_settings WHERE user_id = ?`
const sqlDimention = `SELECT dimentions_id, dimentions_type FROM dimentions WHERE folder_id = ?`
const sqlDimension = `SELECT dimensions_id, dimensions_type FROM dimensions WHERE folder_id = ?`
const sqlFolder = `SELECT budget, bid, creative_id, user_id, folders.status, folders.deleted_at, creative_folder.deleted_at FROM folders LEFT JOIN creative_folder ON folder_id = id WHERE id = ?`
const sqlCreative = `SELECT destination_url, deleted_at FROM creatives cr WHERE cr.id = ?`
const sqlCountries = `SELECT id, iso_2alpha FROM countries`
const sqlNetworks = `SELECT id, pseudonym FROM networks`
const sqlSubNetworks = `SELECT id, pseudonym FROM subnetworks`
const sqlSubNetworkLabels = `SELECT id, label FROM subnetworks`
const sqlBrands = `SELECT id, label FROM brands`
const sqlBrandSlugs = `SELECT id, slug FROM brands`
const sqlVerticals = `SELECT id, label FROM verticals`
const sqlNetworkTypes = `SELECT id, label FROM network_types`
const sqlSubchannels = `SELECT id, label, channel_id FROM subchannels`
const sqlSubnetworkToNetwork = `SELECT id, network_id FROM subnetworks`
const sqlNetworkToNetworkType = `SELECT network_id, network_type_id FROM network_network_type`

type Subchannel struct {
	ChannelID int
	Label     string
}

type Pseudonyms struct {
	Countries  map[string]int
	CountryIDS map[int]string

	Networks           map[string]int
	NetworkIDS         map[int]string
	NetworkTypes       map[string]int
	NetworkTypeIDS     map[int]string
	Subnetworks        map[string]int
	SubnetworkIDS      map[int]string
	SubnetworkLabels   map[string]int
	SubnetworkLabelIDS map[int]string

	Brands       map[string]int
	BrandIDS     map[int]string
	BrandSlugs   map[string]int
	BrandSlugIDS map[int]string
	Verticals    map[string]int
	VerticalIDS  map[int]string

	SubnetworkToNetwork  map[int]int
	NetworkToNetworkType map[int]int

	DeviceTypes   map[string]int
	DeviceTypeIDs map[int]string
	Genders       map[string]int
	GenderIDs     map[int]string

	Subchannels map[Subchannel]int
}

func (c *Pseudonyms) Unmarshal(depth int, env services.BindingDeps) error {
	c.Namespace(env, sqlCountries, &c.Countries, &c.CountryIDS)
	c.Namespace(env, sqlNetworks, &c.Networks, &c.NetworkIDS)
	c.Namespace(env, sqlSubNetworks, &c.Subnetworks, &c.SubnetworkIDS)
	c.Namespace(env, sqlSubNetworkLabels, &c.SubnetworkLabels, &c.SubnetworkLabelIDS)
	c.Namespace(env, sqlBrands, &c.Brands, &c.BrandIDS)
	c.Namespace(env, sqlBrandSlugs, &c.BrandSlugs, &c.BrandSlugIDS)
	c.Namespace(env, sqlVerticals, &c.Verticals, &c.VerticalIDS)
	c.Namespace(env, sqlNetworkTypes, &c.NetworkTypes, &c.NetworkTypeIDS)

	c.SubchannelLoad(env, sqlSubchannels, &c.Subchannels)

	c.Map(env, sqlNetworkToNetworkType, &c.NetworkToNetworkType)
	c.Map(env, sqlSubnetworkToNetwork, &c.SubnetworkToNetwork)

	c.DeviceTypes = map[string]int{"desktop": 1, "mobile": 2, "tablet": 3, "unknown": 4}
	c.DeviceTypeIDs = map[int]string{1: "desktop", 2: "mobile", 3: "tablet", 4: "unknown"}
	c.Genders = map[string]int{"male": 1, "female": 2}
	c.GenderIDs = map[int]string{1: "male", 2: "female"}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), c, tojson(c))
	return nil
}

func (c *Pseudonyms) Map(env services.BindingDeps, sql string, dest *map[int]int) error {
	rows, err := env.ConfigDB.Query(sql)
	if err != nil {
		env.Debug.Println("err", err)
		return err
	}
	*dest = make(map[int]int)
	for rows.Next() {
		var left_side int
		var right_side int
		if err := rows.Scan(&left_side, &right_side); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		(*dest)[left_side] = right_side
	}
	return nil
}

func (c *Pseudonyms) Namespace(env services.BindingDeps, sql string, dest *map[string]int, dest2 *map[int]string) error {
	rows, err := env.ConfigDB.Query(sql)
	if err != nil {
		env.Debug.Println("err", err)
		return err
	}
	*dest = make(map[string]int)
	*dest2 = make(map[int]string)
	for rows.Next() {
		var realName string
		var id int
		if err := rows.Scan(&id, &realName); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		(*dest)[realName] = id
		(*dest2)[id] = realName
	}
	return nil
}

func (c *Pseudonyms) SubchannelLoad(env services.BindingDeps, sql string, dest *map[Subchannel]int) error {
	rows, err := env.ConfigDB.Query(sql)
	if err != nil {
		env.Debug.Println("err", err)
		return err
	}
	*dest = make(map[Subchannel]int)
	for rows.Next() {
		var label string
		var id int
		var channelId int
		if err := rows.Scan(&id, &label, &channelId); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		(*dest)[Subchannel{ChannelID: channelId, Label: label}] = id
	}
	return nil
}

type Users []*User

func (f *Users) ByID(id int) *User {
	for _, u := range *f {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (c *Users) Add(ch *User) int {
	m := 1
	for _, och := range *c {
		if och.ID >= m {
			m = och.ID + 1
		}
	}
	ch.ID = m
	*c = append(*c, ch)
	return ch.ID
}

func (f *Users) Unmarshal(depth int, env services.BindingDeps) error {
	var rows *sql.Rows
	var err error
	rows, err = env.ConfigDB.Query(`SELECT users.id, traffic_status FROM users LEFT JOIN customer ON user_id = users.id`)
	if err != nil {
		env.Debug.Println("err", err)
		return err
	}
	var id, status int
	*f = (*f)[:0]
	for rows.Next() {
		if err := rows.Scan(&id, &status); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		*f = append(*f, &User{ID: id, Status: status})
	}
	for _, f := range *f {
		if err := f.Unmarshal(depth+1, env); err != nil {
			env.Debug.Println("err", err)
			return err
		}
	}
	env.Debug.Printf("LOADED %s %T %s", wide(depth), f, tojson(f))
	return nil
}

type User struct {
	ID     int
	IPs    []string
	Age    int
	Key    string
	B64    *B64
	Status int
}

func (u *User) Unmarshal(depth int, env services.BindingDeps) error {
	rows, err := env.ConfigDB.Query(sqlUserIPs, u.ID)
	if err != nil {
		env.Debug.Println("err", err)
		return err
	}
	var ip string
	for rows.Next() {
		if err := rows.Scan(&ip); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		u.IPs = append(u.IPs, ip)
	}

	{
		rows, err := env.ConfigDB.Query(sqlUser, u.ID)
		if err != nil {
			env.Debug.Println("err", err)
			return err
		}
		var value string
		var setting int
		for rows.Next() {
			if err := rows.Scan(&setting, &value); err != nil {
				env.Debug.Println("err", err)
				return err
			}
			switch setting {
			case 5:
				u.Age, _ = strconv.Atoi(value)
			case 6:
				u.Key = value
			}
		}
	}

	s := strings.Split(env.DefaultKey, ":")
	key, iv := s[0], s[1]
	if u.Key != "" {
		key = u.Key
	}
	u.B64 = &B64{Key: []byte(key), IV: []byte(iv)}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), u, tojson(u))
	return nil
}

func AllIDs(table string, env services.BindingDeps) ([]int, error) {
	rows, err := env.ConfigDB.Query(`SELECT id FROM ` + table)
	if err != nil {
		env.Debug.Println("err", err)
		return nil, err
	}
	var id int
	var ids []int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			env.Debug.Println("err", err)
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

type Dimensions struct {
	FolderID   int
	Dimensions []*Dimension
	mode       int
}

func (d *Dimensions) Unmarshal(depth int, env services.BindingDeps) error {
	sql := sqlDimension
	if d.mode == 1 {
		sql = sqlDimention
	}
	rows, err := env.ConfigDB.Query(sql, d.FolderID)
	if err != nil {
		if d.mode == 0 {
			d.mode = 1
			env.Debug.Println("dimension didn't work, trying dimention")
			return d.Unmarshal(depth, env)
		}
		env.Debug.Println("err", err)
		return err
	}
	for rows.Next() {
		dim := &Dimension{}
		if err := rows.Scan(&dim.Value, &dim.Type); err != nil {
			env.Debug.Println("err", err)
			return err
		}
		d.Dimensions = append(d.Dimensions, dim)
	}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), d, tojson(d))
	return nil
}

type Dimension struct {
	Type  string
	Value int
}

func (d *Dimension) Transfer(f *Folder) error {
	parts := strings.Split(d.Type, `\`)
	switch parts[len(parts)-1] {
	case `Vertical`:
		f.Vertical = append(f.Vertical, d.Value)
		return nil
	case `Country`:
		f.Country = append(f.Country, d.Value)
		return nil
	case `Brand`:
		f.Brand = append(f.Brand, d.Value)
		return nil
	case `Network`:
		f.Network = append(f.Network, d.Value)
		return nil
	case `SubNetwork`:
		f.SubNetwork = append(f.SubNetwork, d.Value)
		return nil
	case `NetworkType`:
		f.NetworkType = append(f.NetworkType, d.Value)
		return nil
	case `Gender`:
		f.Gender = append(f.Gender, d.Value)
		return nil
	case `DeviceType`:
		f.DeviceType = append(f.DeviceType, d.Value)
		return nil
	default:
		return fmt.Errorf(`unknown type: %s`, d.Type)
	}
}

type Folder struct {
	ID       int
	ParentID *int
	Children []int
	Creative []int
	CPC      int
	Budget   int
	OwnerID  int

	Vertical    []int
	Country     []int
	Brand       []int
	Network     []int
	SubNetwork  []int
	NetworkType []int
	Gender      []int
	DeviceType  []int

	Active bool

	mode int
}

func (f *Folder) Unmarshal(depth int, env services.BindingDeps) error {
	// var child_id, creative_id int
	var creative_id sql.NullInt64
	row := env.ConfigDB.QueryRow(sqlFolder, f.ID)

	var budget, bid sql.NullInt64
	var live sql.NullString
	var folder_deleted_at, creative_deleted_at pq.NullTime
	if err := row.Scan(&budget, &bid, &creative_id, &f.OwnerID, &live, &folder_deleted_at, &creative_deleted_at); err != nil {
		if f.mode == 0 {
			f.mode = 1
			env.Debug.Println("users didn't work, trying user")
			return f.Unmarshal(depth, env)
		}
		env.Debug.Println("err", err)
		return err
	}

	if live.Valid {
		if live.String == "live" {
			f.Active = true
		}
	}
	if folder_deleted_at.Valid {
		f.Active = false
	}
	if creative_deleted_at.Valid {
		f.Active = false
	}

	if budget.Valid {
		f.Budget = int(budget.Int64)
	}

	if bid.Valid {
		f.CPC = int(bid.Int64)
	}

	if creative_id.Valid {
		f.Creative = append(f.Creative, int(creative_id.Int64))
	}

	{
		rows, err := env.ConfigDB.Query(`SELECT child_folder_id FROM parent_folder WHERE parent_folder_id = ?`, f.ID)
		if err != nil {
			return err
		}
		var id int
		for rows.Next() {
			if err := rows.Scan(&id); err != nil {
				env.Debug.Println("err", err)
				return err
			}
			f.Children = append(f.Children, id)
		}
	}

	{
		rows, err := env.ConfigDB.Query(`SELECT parent_folder_id FROM parent_folder WHERE child_folder_id = ?`, f.ID)
		if err != nil {
			return err
		}
		var id sql.NullInt64
		for rows.Next() {
			if err := rows.Scan(&id); err != nil {
				env.Debug.Println("err", err)
				return err
			}
			if id.Valid {
				i := int(id.Int64)
				f.ParentID = &i
			}
		}
	}

	// dimensions
	d := &Dimensions{FolderID: f.ID}
	if err := d.Unmarshal(depth+1, env); err != nil {
		env.Debug.Println("err", err)
		return err
	}
	for _, dim := range d.Dimensions {
		if err := dim.Transfer(f); err != nil {
			env.Debug.Println("err", err)
			return err
		}
	}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), f, tojson(f))
	return nil
}

func (f *Folder) String() string {
	dims := fmt.Sprintf(`ve %d, co %d, br %d, ne %d, su %d, nt %d, ge %d, de %d`, f.Vertical, f.Country, f.Brand, f.Network, f.SubNetwork, f.NetworkType, f.Gender, f.DeviceType)
	return fmt.Sprintf(`folder %d (child %d, cpc %d, #cr %d, dims %s)`, f.ID, len(f.Children), f.CPC, len(f.Creative), dims)
}

type Folders []*Folder

func (f *Folders) ByID(id int) *Folder {
	for _, u := range *f {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (c *Folders) Add(ch *Folder) int {
	m := 1
	for _, och := range *c {
		if och.ID >= m {
			m = och.ID + 1
		}
	}
	ch.ID = m
	*c = append(*c, ch)
	for _, child := range ch.Children {
		c.ByID(child).ParentID = &ch.ID
	}
	return ch.ID
}

func (f *Folders) Unmarshal(depth int, env services.BindingDeps) error {
	if ids, err := AllIDs("folders", env); err != nil {
		return err
	} else {
		*f = (*f)[:0]
		for _, id := range ids {
			ch := &Folder{ID: id}
			if err := ch.Unmarshal(depth+1, env); err != nil {
				env.Debug.Println("err", err)
				return err
			}
			*f = append(*f, ch)
		}
	}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), f, tojson(f))
	return nil
}

func (f *Folders) String() string {
	if f == nil {
		return "x0[]"
	}
	str := []string{}
	for _, fo := range *f {
		str = append(str, fo.String())
	}
	return fmt.Sprintf("x%d[%s]", len(*f), strings.Join(str, `,`))
}

type Creatives []*Creative

func (f *Creatives) ByID(id int) *Creative {
	for _, u := range *f {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (c *Creatives) Add(ch *Creative) int {
	m := 1
	for _, och := range *c {
		if och.ID >= m {
			m = och.ID + 1
		}
	}
	ch.ID = m
	*c = append(*c, ch)
	return ch.ID
}

func (f *Creatives) Unmarshal(depth int, env services.BindingDeps) error {
	if ids, err := AllIDs("creatives", env); err != nil {
		return err
	} else {
		*f = (*f)[:0]
		for _, id := range ids {
			ch := &Creative{ID: id}
			if err := ch.Unmarshal(depth+1, env); err != nil {
				env.Debug.Println("err", err)
				return err
			}
			*f = append(*f, ch)
		}
	}

	env.Debug.Printf("LOADED %s %T %s", wide(depth), f, tojson(f))
	return nil
}

type Creative struct {
	ID          int
	RedirectUrl string
	Active      bool
}

func (c *Creative) Unmarshal(depth int, env services.BindingDeps) error {
	var deleted_at pq.NullTime
	row := env.ConfigDB.QueryRow(sqlCreative, c.ID)
	if err := row.Scan(&c.RedirectUrl, &deleted_at); err != nil {
		env.Debug.Println("err", err)
		return err
	}
	c.Active = true
	if deleted_at.Valid {
		c.Active = false
	}
	return nil
}

func (c *Creative) String() string {
	return fmt.Sprintf(`creative %d (%s)`, c.ID, c.RedirectUrl)
}

type StatsDB struct{}

func (StatsDB) allowFailure(sql string, db *sql.DB) {
	createRes, err := db.Exec(sql)
	if err != nil {
		log.Println("expected an err, recieved:", err, ", assuming query has run already")
	} else {
		log.Println("no error returned, must mean this step had effect")
		log.Println(createRes.LastInsertId())
		log.Println(createRes.RowsAffected())
	}
}

func (s StatsDB) Marshal(db *sql.DB) error {
	log.Println("creating purchases table")
	s.allowFailure(sqlCreatePurchases, db)
	return nil
}

type Recalls struct {
	Env services.BindingDeps
}

func (s Recalls) Save(f json.Marshaler, idLoc *int) error {
	js, _ := f.MarshalJSON()
	var err error
	*idLoc, err = s.Env.Redis.FindID(string(js))
	return err
}

func (s Recalls) Fetch(f json.Unmarshaler, recall string) error {
	target, err := s.Env.Redis.Load(recall)
	if err != nil {
		return err
	}

	if e := f.UnmarshalJSON([]byte(target)); e != nil {
		return e
	}

	return nil
}

type Purchases struct {
	Env      services.BindingDeps
	SkipWork bool
}

func (s Purchases) Save(fs [][17]interface{}, quit func(error) bool) {
	q := []string{}
	args := []interface{}{}

	n := 1
	for _, f := range fs {
		thisInsertString := []string{}
		for range f {
			thisInsertString = append(thisInsertString, fmt.Sprintf(`$%d`, n))
			n++
		}
		q = append(q, "("+strings.Join(thisInsertString, ",")+")")
		args = append(args, f[:]...)
	}

	query := sqlInsertPurchases + strings.Join(q, ",")
	s.Env.Logger.Println("query:", query)

	for attempt := 15; attempt > 0; attempt-- {
		if _, err := s.Env.StatsDB.Exec(query, args...); quit(services.ErrDatabaseMissing{"purchases", err}) {
			if attempt == 0 {
				return
			} else {
				s.Env.Logger.Println("failed, waiting 1 min to try again")
				time.Sleep(time.Minute)
			}
		} else {
			break
		}
	}
}

const sqlInsertPurchases = `INSERT INTO purchases (sale_id, billable, rev_tx, rev_tx_home, rev_ssp, rev_ssp_home, ssp_id, folder_id, creative_id, country_id, vertical_id, brand_id, network_id, subnetwork_id, networktype_id, gender_id, devicetype_id) VALUES `

const sqlCreatePurchases = `CREATE TABLE purchases (
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	sale_id int NOT NULL,
	billable bool NOT NULL,
	rev_tx_home int NOT NULL,
	rev_tx int NOT NULL,
	rev_ssp int NOT NULL,
	rev_ssp_home int NOT NULL,
	ssp_id int NOT NULL,

  	folder_id int NOT NULL,
  	creative_id int NOT NULL,

	country_id int NOT NULL,
	vertical_id int NOT NULL,
	brand_id int NOT NULL,
	network_id int NOT NULL,
	subnetwork_id int NOT NULL,
	networktype_id int NOT NULL,
	gender_id int NOT NULL,
	devicetype_id int NOT NULL
);`
