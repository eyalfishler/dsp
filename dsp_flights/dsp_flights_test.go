package dsp_flights

import (
	"encoding/json"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/clixxa/dsp/bindings"
	"github.com/clixxa/dsp/rtb_types"
	"github.com/clixxa/dsp/services"
	"testing"
)

func TestStageFindClient(t *testing.T) {
	l, fin := bindings.BufferedLogger(t)
	flight := &DemandFlight{}
	flight.Runtime.Logger = l
	flight.Runtime.Logger.Println("testing StoreFlight, before:", flight)
	flight.Runtime.DefaultB64 = &bindings.B64{Key: []byte("gekk"), IV: []byte("whatwhat")}

	store := &flight.Runtime.Storage
	store.Recalls = func(df json.Marshaler, b *int) error {
		t.Log("recall save", df)
		return nil
	}
	flight.Runtime.Logic = SimpleLogic{}

	crid := store.Creatives.Add(&bindings.Creative{})
	own := store.Users.Add(&bindings.User{Age: 10})

	bfid := store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Brand: []int{6}, Creative: []int{crid}, CPC: 350})
	store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, Children: []int{bfid}, CPC: 500})
	store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{4}, CPC: 500})
	store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, Brand: []int{6}, CPC: 50})
	badfolder := store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, CPC: 50})
	store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, CPC: 700, Children: []int{badfolder}})
	randpick := store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, Brand: []int{6}, CPC: 500, Creative: []int{crid}})
	_ = randpick
	store.Folders.Add(&bindings.Folder{OwnerID: own, Active: true, Country: []int{3}, Brand: []int{6}, CPC: 250})

	flight.Raw.Impressions = []rtb_types.Impression{{}}
	flight.Dims.CountryID = 3
	flight.Dims.BrandID = 6

	res := map[int]int{}
	for i := 0; i < 255; i++ {
		flight.Raw.Random255 = i
		flight.Response.SeatBids = nil
		flight.FolderID = 0
		flight.CreativeID = 0
		flight.FullPrice = 0

		flight.Runtime.Logger.Println("testing FindClient, before:", flight)
		FindClient(flight)
		flight.Runtime.Logger.Println("after:", flight)
		fin()
		if _, found := res[flight.FolderID]; !found {
			res[flight.FolderID] = 0
		}
		res[flight.FolderID] += 1
	}
	t.Log(res)
	if d := res[bfid] - res[randpick]; d < -5 || d > 5 {
		t.Error("unequal distribution")
	}
}

func TestWhitelist(t *testing.T) {
	l, fin := bindings.BufferedLogger(t)
	flight := &DemandFlight{}
	flight.Runtime.Logic = SimpleLogic{}
	flight.Runtime.Logger = l
	store := &flight.Runtime.Storage
	f := flight.Runtime.Storage.Folders.ByID(store.Folders.Add(&bindings.Folder{OwnerID: store.Users.Add(&bindings.User{}), Active: true, Creative: []int{store.Creatives.Add(&bindings.Creative{Active: true})}, Network: []int{1, 2}}))
	flight.Dims.NetworkID = 2
	FindClient(flight)
	if flight.FolderID != f.ID {
		t.Error("wrong folder selected, wanted", f.ID, "got", flight.FolderID)
	}
	fin()
}

func TestLoadAll(t *testing.T) {
	db, sqlm, _ := sqlmock.New()

	sqlm.ExpectExec("purchases").WillReturnError(fmt.Errorf(`expectedErr`))

	sqlm.ExpectQuery("folders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))

	sqlm.ExpectQuery("folders").WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"budget", "bid", "creative_id", "owner", "status", "d1", "d2"}).
			AddRow(100, 50, 30, 5, "live", nil, nil))
	sqlm.ExpectQuery("parent_folder").WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("7"))
	sqlm.ExpectQuery("parent_folder").WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("8"))
	sqlm.ExpectQuery("dimentions").WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow(1, "Network").AddRow(2, "Network"))

	sqlm.ExpectQuery("creatives").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
	sqlm.ExpectQuery("creatives").WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"url", "d"}).AddRow("test.com", nil))

	sqlm.ExpectQuery("users").WillReturnRows(sqlmock.NewRows([]string{"id", "d"}).AddRow(5, 0))
	sqlm.ExpectQuery("ip_histories").WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"ip"}).AddRow("1.1.1.1"))
	sqlm.ExpectQuery("user_settings").WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"setting", "value"}).AddRow(6, "what"))

	sqlm.ExpectQuery("SELECT (.+) FROM countries").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM networks").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM subnetworks").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM subnetworks").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM brands").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM verticals").
		WillReturnRows(sqlmock.NewRows([]string{"id", "iso_2alpha"}))

	sqlm.ExpectQuery("SELECT (.+) FROM subchannels").
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel", "label"}))

	sqlm.MatchExpectationsInOrder(false)

	out, dump := bindings.BufferedLogger(t)
	be := &BidEntrypoint{BindingDeps: services.BindingDeps{ConfigDB: db, StatsDB: db, Logger: out, Debug: out, DefaultKey: ":", Redis: &services.RandomCache{&services.CountingCache{}}}}
	be.Cycle(func(err error) bool {
		t.Log("err cycling", err.Error())
		return false
	})
	dump()
	if be.DemandFlight().Runtime.Storage.Folders.ByID(5).Network[1] != 2 {
		t.Error("missing second network in folder whitelist")
	}
	if err := sqlm.ExpectationsWereMet(); err != nil {
		t.Error("err", err.Error())
	}
}
