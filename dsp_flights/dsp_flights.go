package dsp_flights

import (
	"fmt"
	"github.com/clixxa/dsp/bindings"
	"github.com/clixxa/dsp/rtb_types"
	"github.com/clixxa/dsp/services"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Uses environment variables and real database connections to create Runtimes
type BidEntrypoint struct {
	demandFlight atomic.Value

	BindingDeps services.BindingDeps
	Logic       BiddingLogic
	AllTest     bool
}

func (e *BidEntrypoint) Cycle(quit func(error) bool) {
	// create template demand flight
	df := &DemandFlight{}
	if old, found := e.demandFlight.Load().(*DemandFlight); found {
		e.BindingDeps.Debug.Println("using old runtime")
		df.Runtime = old.Runtime
	} else {
		df.Runtime.Logger = e.BindingDeps.Logger
		df.Runtime.Logger.Println("brand new runtime")
		df.Runtime.Debug = e.BindingDeps.Debug
		s := strings.Split(e.BindingDeps.DefaultKey, ":")
		if len(s) != 2 {
			if quit(services.ErrParsing{"encryption key", fmt.Errorf(`missing encryption key...`)}) {
				return
			}
		} else {
			key, iv := s[0], s[1]
			df.Runtime.DefaultB64 = &bindings.B64{Key: []byte(key), IV: []byte(iv)}
		}
		df.Runtime.Logic = e.Logic
		df.Runtime.TestOnly = e.AllTest

		if e.BindingDeps.StatsDB != nil {
			if quit(services.ErrParsing{"stats initial marshal", (bindings.StatsDB{}).Marshal(e.BindingDeps.StatsDB)}) {
				return
			}
		}
	}

	if e.BindingDeps.ConfigDB != nil {
		if quit(services.ErrParsing{"config Folders", df.Runtime.Storage.Folders.Unmarshal(1, e.BindingDeps)}) {
			return
		}
		if quit(services.ErrParsing{"config Creatives", df.Runtime.Storage.Creatives.Unmarshal(1, e.BindingDeps)}) {
			return
		}

		if quit(services.ErrParsing{"config Users", df.Runtime.Storage.Users.Unmarshal(1, e.BindingDeps)}) {
			return
		}
		if quit(services.ErrParsing{"config Pseudonyms", df.Runtime.Storage.Pseudonyms.Unmarshal(1, e.BindingDeps)}) {
			return
		}
	}

	e.demandFlight.Store(df)
}

func (e *BidEntrypoint) DemandFlight() *DemandFlight {
	sf := e.demandFlight.Load().(*DemandFlight)
	flight := &DemandFlight{}
	flight.Runtime = sf.Runtime
	return flight
}

type BiddingLogic interface {
	SelectFolderAndCreative(flight *DemandFlight, folders []ElegibleFolder, totalCpc int)
	CalculateRevshare(flight *DemandFlight) float64
}

type SimpleLogic struct {
}

func (s SimpleLogic) SelectFolderAndCreative(flight *DemandFlight, folders []ElegibleFolder, totalCpc int) {
	eg := folders[flight.Raw.Random255%len(folders)]
	foldIds := make([]string, len(folders))
	for n, folder := range folders {
		foldIds[n] = strconv.Itoa(folder.FolderID)
	}
	flight.Runtime.Logger.Println(`folders`, strings.Join(foldIds, ","), `to choose from, picked`, eg.FolderID)
	flight.FolderID = eg.FolderID
	flight.FullPrice = eg.BidAmount
	folder := flight.Runtime.Storage.Folders.ByID(eg.FolderID)
	flight.CreativeID = folder.Creative[flight.Raw.Random255%len(folder.Creative)]
}

func (s SimpleLogic) CalculateRevshare(flight *DemandFlight) float64 { return 98.0 }

type DemandFlight struct {
	Runtime struct {
		DefaultB64 *bindings.B64
		Storage    struct {
			Folders    bindings.Folders
			Creatives  bindings.Creatives
			Pseudonyms bindings.Pseudonyms
			Users      bindings.Users
		}
		Logger   *log.Logger
		Debug    *log.Logger
		TestOnly bool
		Logic    BiddingLogic
	}

	FullPrice int
	StartTime time.Time
	Error     error

	rtb_types.BidSnapshot
	rtb_types.WinNotice
}

func (df *DemandFlight) String() string {
	e := "nil"
	if df.Error != nil {
		e = df.Error.Error()
	}
	return fmt.Sprintf(`demandflight e%s`, e)
}

func (df *DemandFlight) Launch() {
	defer func() {
		if err := recover(); err != nil {
			df.Runtime.Logger.Println("uncaught panic, stack trace following", err)
			s := debug.Stack()
			df.Runtime.Logger.Println(string(s))
		}
	}()
	FindClient(df)
	PrepareResponse(df)
}

// Fill out the elegible bid
func FindClient(flight *DemandFlight) {
	flight.Runtime.Logger.Println(`starting FindClient`, flight.String())
	if flight.Error != nil {
		return
	}

	FolderMatches := func(folder *bindings.Folder) string {
		if !folder.Active {
			return "Inactive"
		}
		u := flight.Runtime.Storage.Users.ByID(folder.OwnerID)
		if u.Status != 0 {
			return "UserStatus=" + strconv.Itoa(u.Status)
		}
		if flight.Raw.Test {
			goto CheckBrand
		}

		if len(folder.Country) > 0 {
			for _, c := range folder.Country {
				if flight.Dims.CountryID == c {
					goto CheckBrand
				}
			}
			return "Country"
		}
	CheckBrand:
		if len(folder.Brand) > 0 {
			for _, v := range folder.Brand {
				if flight.Dims.BrandID == v {
					goto CheckNetwork
				}
			}
			return "Brand"
		}
	CheckNetwork:
		if len(folder.Network) > 0 {
			for _, v := range folder.Network {
				if flight.Dims.NetworkID == v {
					goto CheckNetworkType
				}
			}
			return "Network"
		}
	CheckNetworkType:
		if len(folder.NetworkType) > 0 {
			for _, v := range folder.NetworkType {
				if flight.Dims.NetworkTypeID == v {
					goto CheckSubNetwork
				}
			}
			return "NetworkType"
		}
	CheckSubNetwork:
		if len(folder.SubNetwork) > 0 {
			for _, v := range folder.SubNetwork {
				if flight.Dims.SubNetworkID == v {
					goto CheckGender
				}
			}
			return "SubNetwork"
		}
	CheckGender:
		if len(folder.Gender) > 0 {
			for _, v := range folder.Gender {
				if flight.Dims.GenderID == v {
					goto CheckDeviceType
				}
			}
			return "Gender"
		}
	CheckDeviceType:
		if len(folder.DeviceType) > 0 {
			for _, v := range folder.DeviceType {
				if flight.Dims.DeviceTypeID == v {
					goto CheckVertical
				}
			}
			return "DeviceType"
		}
	CheckVertical:
		if len(folder.Vertical) > 0 {
			for _, v := range folder.Vertical {
				if flight.Dims.VerticalID == v {
					goto CheckBidfloor
				}
			}
			return "Vertical"
		}
	CheckBidfloor:
		if folder.CPC > 0 && folder.CPC < flight.Raw.Impressions[0].BidFloor {
			return "CPC"
		}
		return ""
	}

	folders := []ElegibleFolder{}
	totalCpc := 0

	Visit := func(folder *bindings.Folder) bool {
		if s := FolderMatches(folder); s != "" {
			flight.Runtime.Logger.Printf("folder %d doesn't match cause %s..", folder.ID, s)
			return false
		}

		flight.Runtime.Logger.Printf("folder %d matches..", folder.ID)

		found := false
		for _, c := range folder.Creative {
			cr := flight.Runtime.Storage.Creatives.ByID(c)
			if cr.Active {
				found = true
				break
			}
		}

		if found {
			cpc := folder.CPC
			if folder.ParentID != nil && cpc == 0 {
				cpc = flight.Runtime.Storage.Folders.ByID(*folder.ParentID).CPC
			}
			totalCpc += cpc
			folders = append(folders, ElegibleFolder{FolderID: folder.ID, BidAmount: cpc})
		}

		return true
	}

	for _, folder := range flight.Runtime.Storage.Folders {
		if folder.ParentID == nil {
			if !Visit(folder) {
				continue
			}
			for _, r := range folder.Children {
				if !Visit(flight.Runtime.Storage.Folders.ByID(r)) {
					continue
				}
			}
		}
	}

	if len(folders) == 0 {
		flight.Runtime.Logger.Println(`no folder found`)
		return
	}

	flight.Runtime.Logic.SelectFolderAndCreative(flight, folders, totalCpc)
}

func PrepareResponse(flight *DemandFlight) {
	if flight.FolderID == 0 {
		return
	}
	revShare := flight.Runtime.Logic.CalculateRevshare(flight)
	if revShare > 100 {
		revShare = 100
	}
	fp := float64(flight.FullPrice)
	flight.Runtime.Logger.Printf("rev calculated at %f", revShare)
	price := fp * revShare / 100
	flight.ExtraInfo = &flight.BidSnapshot
	flight.Margin = flight.FullPrice - int(price)
	flight.OfferedPrice = flight.FullPrice - flight.Margin
	cr := flight.Runtime.Storage.Creatives.ByID(flight.CreativeID)
	flight.URL = cr.RedirectUrl

	if flight.Error != nil {
		flight.Runtime.Logger.Println(`error occured in FindClient: %s`, flight.Error.Error())
		return
	}
	flight.Runtime.Logger.Println("finished FindClient", flight.String())
}

type ElegibleFolder struct {
	FolderID  int
	BidAmount int
}
