// Package addinfo implements the profile AddInfo flow for s557 accounts.
// After a successful verify, this updates city, hometown, school, and relationship status.
//
// City/Hometown: uses Facebook's own inference from ViewNuxQuery (zero config needed).
// School: loads school name from Config/AddInfo/schools.txt, searches for Page ID via API.
package addinfo

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"

	"HVRIns/internal/instagram"
)

// Result holds which fields were successfully updated.
type Result struct {
	CitySet         bool
	HometownSet     bool
	SchoolSet       bool
	CollegeSet      bool
	WorkSet         bool
	RelationshipSet bool
	Notes           []string
}

// RunAddInfo updates profile info for a live s557 account.
// City/Hometown are set automatically using Facebook's own location inference — no config needed.
// School requires Config/AddInfo/schools.txt with school names (one per line).
func RunAddInfo(ctx context.Context, session *instagram.Session, cfg *instagram.AddInfoConfig, notify func(string)) *Result {
	res := &Result{}
	if cfg == nil || !cfg.Enabled {
		return res
	}

	token := session.Token
	if token == "" {
		notify("[AddInfo] No access token, skip")
		return res
	}

	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "Config/AddInfo"
	}
	delayMs := cfg.DelayMs
	if delayMs <= 0 {
		delayMs = 2000
	}

	uid := session.UID
	locale := extractLocale(session.Cookie)
	machineID := session.Datr
	if machineID == "" {
		machineID = "Kb_UaVc0y5UrH8GU29y9f_9c"
	}
	deviceID := session.DeviceID
	if deviceID == "" {
		deviceID = newUUID()
	}
	sessID := "UFS-" + newUUID() + "-fg-1"

	cl, err := newClient(session.Proxy)
	if err != nil {
		notify(fmt.Sprintf("[AddInfo] Client error: %v", err))
		return res
	}
	defer cl.CloseIdleConnections()

	notify("[AddInfo] Init profile wizard...")
	if err := doViewMutation(ctx, cl, token, uid, locale, machineID, deviceID, sessID); err != nil {
		notify(fmt.Sprintf("[AddInfo] ViewMutation warning: %v", err))
	}
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	if err := doStartMutation(ctx, cl, token, uid, locale, machineID, deviceID, sessID); err != nil {
		notify(fmt.Sprintf("[AddInfo] StartMutation warning: %v", err))
	}
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	mutID := 3

	// City — uses FB's own inference (zero config, no file needed).
	if cfg.City {
		notify("[AddInfo] Setting current city (using FB inference)...")
		if err := saveField(ctx, cl, token, uid, locale, machineID, deviceID, sessID, fieldCurrentCity, "", mutID, notify); err != nil {
			notify(fmt.Sprintf("[AddInfo] City: %v", err))
		} else {
			res.CitySet = true
			res.Notes = append(res.Notes, "city=auto")
			notify("[AddInfo] City set")
			mutID++
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Hometown — FB inference is always empty for this field; search by city name from file.
	if cfg.Hometown {
		if hometownPageID, hometownName, err := pickAndSearchCity(ctx, cl, token, locale, machineID, deviceID, dataDir, notify); err != nil {
			notify(fmt.Sprintf("[AddInfo] Hometown: %v", err))
		} else {
			if err := saveField(ctx, cl, token, uid, locale, machineID, deviceID, sessID, fieldHometown, hometownPageID, mutID, notify); err != nil {
				notify(fmt.Sprintf("[AddInfo] Hometown save: %v", err))
			} else {
				res.HometownSet = true
				res.Notes = append(res.Notes, "hometown="+hometownName)
				notify(fmt.Sprintf("[AddInfo] Hometown set: %s", hometownName))
				mutID++
			}
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// School — direct mutation (ProfileEditEducationExperienceSaveMutation), no NUX wizard needed.
	if cfg.School {
		if schoolPageID, schoolName, err := pickAndSearchSchool(ctx, cl, token, locale, machineID, deviceID, dataDir, notify); err != nil {
			notify(fmt.Sprintf("[AddInfo] School: %v", err))
		} else {
			if err := saveSchoolDirect(ctx, cl, token, uid, locale, machineID, deviceID, schoolPageID, schoolName, mutID, notify); err != nil {
				notify(fmt.Sprintf("[AddInfo] School save: %v", err))
			} else {
				res.SchoolSet = true
				res.Notes = append(res.Notes, "school="+schoolName)
				notify(fmt.Sprintf("[AddInfo] School set: %s", schoolName))
				mutID++
			}
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// College — search by name from colleges.txt.
	if cfg.College {
		if collegePageID, collegeName, err := pickAndSearchCollege(ctx, cl, token, locale, machineID, deviceID, dataDir, notify); err != nil {
			notify(fmt.Sprintf("[AddInfo] College: %v", err))
		} else {
			if err := saveField(ctx, cl, token, uid, locale, machineID, deviceID, sessID, fieldCollege, collegePageID, mutID, notify); err != nil {
				notify(fmt.Sprintf("[AddInfo] College save: %v", err))
			} else {
				res.CollegeSet = true
				res.Notes = append(res.Notes, "college="+collegeName)
				notify(fmt.Sprintf("[AddInfo] College set: %s", collegeName))
				mutID++
			}
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Work — direct mutation (ProfileEditWorkViewMutation), no NUX wizard needed.
	if cfg.Work {
		if workPageID, workName, err := pickAndSearchWork(ctx, cl, token, locale, machineID, deviceID, dataDir, notify); err != nil {
			notify(fmt.Sprintf("[AddInfo] Work: %v", err))
		} else {
			if err := saveWorkDirect(ctx, cl, token, uid, locale, machineID, deviceID, workPageID, workName, mutID, notify); err != nil {
				notify(fmt.Sprintf("[AddInfo] Work save: %v", err))
			} else {
				res.WorkSet = true
				res.Notes = append(res.Notes, "work="+workName)
				notify(fmt.Sprintf("[AddInfo] Work set: %s", workName))
				mutID++
			}
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Relationship — set to Single (status=1).
	if cfg.Relationship {
		notify("[AddInfo] Setting relationship: single...")
		if err := doRelationshipMutation(ctx, cl, token, uid, locale, machineID, deviceID, sessID, 1, mutID); err != nil {
			notify(fmt.Sprintf("[AddInfo] Relationship: %v", err))
		} else {
			res.RelationshipSet = true
			res.Notes = append(res.Notes, "relationship=single")
			notify("[AddInfo] Relationship set")
		}
	}

	return res
}

// pickAndSearchCity picks a random city name from hometowns.txt and searches for its FB Page ID.
func pickAndSearchCity(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, dataDir string, notify func(string)) (pageID, name string, err error) {
	entries, loadErr := loadEntries(filepath.Join(dataDir, "hometowns.txt"))
	if loadErr != nil {
		return "", "", fmt.Errorf("load hometowns.txt: %w", loadErr)
	}
	if len(entries) == 0 {
		return "", "", fmt.Errorf("hometowns.txt is empty — add city names (one per line)")
	}

	picked, _ := pickRandom(entries)
	notify(fmt.Sprintf("[AddInfo] Searching city: %s", picked.Name))

	pageID, foundName, searchErr := searchCity(ctx, cl, token, locale, machineID, deviceID, picked.Name)
	if searchErr != nil {
		return "", "", fmt.Errorf("city search API error [%s]: %w", picked.Name, searchErr)
	}
	if pageID == "" {
		return "", "", fmt.Errorf("city not found (no FB page): %s", picked.Name)
	}
	return pageID, foundName, nil
}

// pickAndSearchCollege picks a random university name from colleges.txt and searches for its FB Page ID.
func pickAndSearchCollege(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, dataDir string, notify func(string)) (pageID, name string, err error) {
	entries, loadErr := loadEntries(filepath.Join(dataDir, "colleges.txt"))
	if loadErr != nil {
		return "", "", fmt.Errorf("load colleges.txt: %w", loadErr)
	}
	if len(entries) == 0 {
		return "", "", fmt.Errorf("colleges.txt is empty — add university names (one per line)")
	}
	picked, _ := pickRandom(entries)
	notify(fmt.Sprintf("[AddInfo] Searching college: %s", picked.Name))
	pageID, foundName, searchErr := searchCollege(ctx, cl, token, locale, machineID, deviceID, picked.Name)
	if searchErr != nil {
		return "", "", fmt.Errorf("college search API error [%s]: %w", picked.Name, searchErr)
	}
	if pageID == "" {
		return "", "", fmt.Errorf("college not found (no FB page): %s", picked.Name)
	}
	return pageID, foundName, nil
}

// pickAndSearchWork picks a random company name from companies.txt and searches for its FB Page ID.
func pickAndSearchWork(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, dataDir string, notify func(string)) (pageID, name string, err error) {
	entries, loadErr := loadEntries(filepath.Join(dataDir, "companies.txt"))
	if loadErr != nil {
		return "", "", fmt.Errorf("load companies.txt: %w", loadErr)
	}
	if len(entries) == 0 {
		return "", "", fmt.Errorf("companies.txt is empty — add company names (one per line)")
	}
	picked, _ := pickRandom(entries)
	notify(fmt.Sprintf("[AddInfo] Searching work: %s", picked.Name))
	pageID, foundName, searchErr := searchWork(ctx, cl, token, locale, machineID, deviceID, picked.Name)
	if searchErr != nil {
		return "", "", fmt.Errorf("work search API error [%s]: %w", picked.Name, searchErr)
	}
	if pageID == "" {
		return "", "", fmt.Errorf("work not found (no FB page): %s", picked.Name)
	}
	return pageID, foundName, nil
}

// pickAndSearchSchool picks a random school name from schools.txt and searches for its FB Page ID.
func pickAndSearchSchool(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, dataDir string, notify func(string)) (pageID, name string, err error) {
	entries, loadErr := loadEntries(filepath.Join(dataDir, "schools.txt"))
	if loadErr != nil {
		return "", "", fmt.Errorf("load schools.txt: %w", loadErr)
	}
	if len(entries) == 0 {
		return "", "", fmt.Errorf("schools.txt is empty — add school names (one per line)")
	}

	picked, _ := pickRandom(entries)
	notify(fmt.Sprintf("[AddInfo] Searching school: %s", picked.Name))

	pageID, foundName, searchErr := searchSchool(ctx, cl, token, locale, machineID, deviceID, picked.Name)
	if searchErr != nil {
		return "", "", fmt.Errorf("school search API error [%s]: %w", picked.Name, searchErr)
	}
	if pageID == "" {
		return "", "", fmt.Errorf("school not found (no FB page): %s", picked.Name)
	}
	return pageID, foundName, nil
}

func extractLocale(cookieStr string) string {
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "locale=") {
			return strings.TrimPrefix(part, "locale=")
		}
	}
	return "en_US"
}
