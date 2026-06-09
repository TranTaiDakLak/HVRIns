package addinfo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
)

const (
	graphURL = "https://graph.facebook.com/graphql"

	docViewMutation         = "9195645420564568"
	docStartMutation        = "9616940285057275"
	docViewNuxQuery         = "9399351610192340"
	docSaveMutation         = "29286669860948214"
	docRelationshipMutation = "9773237786048037"
	docSchoolSearch         = "30413951324870438"
	docCitySearch           = "24170897439179750"

	// Direct edit mutations — bypass NUX wizard, confirmed working in VerifyCloneVIP.
	docSchoolSave = "2228867157143096" // ProfileEditEducationExperienceSaveMutation
	docWorkSave   = "6372983786060488" // ProfileEditWorkViewMutation
)

type fieldType struct {
	stepType string
	category string
	n        int // field index used in profile_question_id base64
}

var (
	fieldCurrentCity = fieldType{"CURRENT_CITY_PROFILE_FIELD", "CURRENT_CITY", 3}
	fieldHometown    = fieldType{"HOMETOWN_PROFILE_FIELD", "HOMETOWN", 2}
	fieldHighSchool  = fieldType{"HIGH_SCHOOL_PROFILE_FIELD", "HIGH_SCHOOL", 1}
	fieldCollege     = fieldType{"COLLEGE_PROFILE_FIELD", "COLLEGE", 4}
	fieldWork        = fieldType{"WORK_PROFILE_FIELD", "WORK_CURRENT", 5}
)

func doViewMutation(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, sessID string) error {
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"actor_id":           uid,
			"client_mutation_id": "1",
		},
	}
	return callRelayMutation(ctx, cl, token, locale, machineID, deviceID, sessID,
		"ProfileWizardNuxMutationViewMutation", docViewMutation, vars, true)
}

func doStartMutation(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, sessID string) error {
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"actor_id":           uid,
			"client_mutation_id": "2",
		},
	}
	return callRelayMutation(ctx, cl, token, locale, machineID, deviceID, sessID,
		"ProfileWizardNuxMutationStartMutation", docStartMutation, vars, true)
}

// saveField fetches the NUX session + inference, then saves the profile field.
// optionID is the fallback Page ID; if empty, the inference from FB is required.
// Returns the notify-ready log line prefix for the caller to print.
func saveField(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, sessID string, ft fieldType, optionID string, mutID int, notify func(string)) error {
	sess, inferenceID, err := fetchNuxSession(ctx, cl, token, uid, locale, machineID, deviceID, ft)
	if err != nil {
		return fmt.Errorf("viewNuxQuery: %w", err)
	}

	// For fields with a searched page ID, prefer the supplied option; otherwise use FB inference.
	pageID := optionID
	if optionID == "" && inferenceID != "" {
		pageID = inferenceID
	}
	if pageID == "" {
		return fmt.Errorf("no page ID available (inference empty, no search result)")
	}

	notify(fmt.Sprintf("[AddInfo] Saving %s pageID=%s mutID=%d", ft.stepType, pageID, mutID))

	profileQID := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("profile_question:%s_%d", uid, ft.n)),
	)

	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"actor_id":                   uid,
			"client_mutation_id":         fmt.Sprintf("%d", mutID),
			"privacy":                    privacyEveryone(),
			"profile_question_id":        profileQID,
			"profile_question_option_id": pageID,
			"session":                    sess,
			"logging_data":               map[string]interface{}{},
		},
	}

	resp, err := callSaveMutation(ctx, cl, token, locale, machineID, deviceID, sessID, vars)
	if err != nil {
		return err
	}

	// Detect semantic FB errors (HTTP 200 but contains "errors" array).
	if strings.Contains(resp, `"errors"`) {
		snippet := resp
		if len(snippet) > 250 {
			snippet = snippet[:250]
		}
		return fmt.Errorf("FB returned error: %s", snippet)
	}
	notify(fmt.Sprintf("[AddInfo] Save OK: %s → resp: %s", ft.stepType, truncate(resp, 80)))
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func doRelationshipMutation(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, sessID string, status, mutID int) error {
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"actor_id":           uid,
			"client_mutation_id": fmt.Sprintf("%d", mutID),
			"status":             status,
			"privacy":            privacyEveryone(),
		},
	}
	return callRelayMutation(ctx, cl, token, locale, machineID, deviceID, sessID,
		"ProfileEditRelationshipViewMutation", docRelationshipMutation, vars, true)
}

// searchSchool searches for a high school Page ID by name.
func searchSchool(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, name string) (string, string, error) {
	return searchEntity(ctx, cl, token, locale, machineID, deviceID, name,
		[]string{"HIGH_SCHOOL"}, "hub_education")
}

// searchCollege searches for a university/college Page ID by name.
func searchCollege(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, name string) (string, string, error) {
	return searchEntity(ctx, cl, token, locale, machineID, deviceID, name,
		[]string{"UNIVERSITY", "EDUCATION_COMPANY", "UNIVERSITY_STATUS"}, "hub_college")
}

// searchWork searches for a company/workplace Page ID by name.
func searchWork(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, name string) (string, string, error) {
	return searchEntity(ctx, cl, token, locale, machineID, deviceID, name,
		[]string{"COMPANY"}, "hub_work")
}

// searchEntity is the generic typeahead search for school/college/work.
// Tries progressively shorter queries if the full name returns no results.
func searchEntity(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, entityName string, categories []string, context string) (string, string, error) {
	queries := buildFallbackQueries(entityName)
	for _, q := range queries {
		id, name, err := doEntitySearch(ctx, cl, token, locale, machineID, deviceID, q, categories, context)
		if err != nil {
			return "", "", err
		}
		if id != "" {
			return id, name, nil
		}
	}
	return "", "", nil
}

func doEntitySearch(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, query string, categories []string, searchContext string) (string, string, error) {
	vars := map[string]interface{}{
		"queryString": query,
		"categories":  categories,
		"context":     searchContext,
		"maxResults":  10,
	}
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "ProfileEditTypeaheadGraphQLSearchSourceQuery", docSchoolSearch, string(varsJSON), "", false)
	headers := relayHeaders("ProfileEditTypeaheadGraphQLSearchSourceQuery", machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)

	respStr, err := doPost(ctx, cl, targetURL, body, headers)
	if err != nil {
		return "", "", fmt.Errorf("entity search HTTP: %w", err)
	}
	return parseSchoolSearchResult(respStr)
}

// buildFallbackQueries returns [fullName, first2Words, firstWord] deduplicated.
func buildFallbackQueries(name string) []string {
	words := strings.Fields(name)
	seen := map[string]bool{}
	var out []string
	candidates := []string{
		name,
		strings.Join(words[:min2(len(words), 2)], " "),
		words[0],
	}
	for _, c := range candidates {
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type schoolSearchResp struct {
	Data struct {
		EntitiesNamed struct {
			SearchResults struct {
				Edges []struct {
					Node struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"search_results"`
		} `json:"entities_named"`
	} `json:"data"`
}

func parseSchoolSearchResult(respStr string) (id, name string, err error) {
	var r schoolSearchResp
	if jsonErr := json.Unmarshal([]byte(respStr), &r); jsonErr != nil {
		snippet := respStr
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		return "", "", fmt.Errorf("parse school search: %w | resp: %s", jsonErr, snippet)
	}
	edges := r.Data.EntitiesNamed.SearchResults.Edges
	if len(edges) == 0 {
		return "", "", nil
	}
	return edges[0].Node.ID, edges[0].Node.Name, nil
}

// fetchNuxSession calls ViewNuxQuery and returns the session token + first inference page ID.
// inferenceID may be empty if FB has no location suggestion for the account.
func fetchNuxSession(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID string, ft fieldType) (sess, inferenceID string, err error) {
	vars := map[string]interface{}{
		"profileFieldStepType":         []string{ft.stepType},
		"profileQuestionCategories":    []string{ft.category},
		"suggestionProfilePictureSize": 144,
	}
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "ProfileWizardProfileFieldViewNuxQuery", docViewNuxQuery, string(varsJSON), "", false)

	headers := relayHeaders("ProfileWizardProfileFieldViewNuxQuery", machineID, deviceID,
		[2]string{"privacy_context", "ProfileNuxRoute"},
	)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)

	respStr, err := doPost(ctx, cl, targetURL, body, headers)
	if err != nil {
		return "", "", fmt.Errorf("HTTP: %w", err)
	}
	return parseNuxSession(respStr)
}

type viewNuxResp struct {
	Data struct {
		Viewer struct {
			ProfileQuestions struct {
				Edges []struct {
					Session string `json:"session"`
					Node    struct {
						Inferences struct {
							Edges []struct {
								Node struct {
									Page struct {
										ID   string `json:"id"`
										Name string `json:"name"`
									} `json:"page"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"inferences"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"profile_questions"`
		} `json:"viewer"`
	} `json:"data"`
}

func parseNuxSession(respStr string) (sess, inferenceID string, err error) {
	var r viewNuxResp
	if jsonErr := json.Unmarshal([]byte(respStr), &r); jsonErr != nil {
		return "", "", fmt.Errorf("parse ViewNuxQuery response: %w", jsonErr)
	}
	edges := r.Data.Viewer.ProfileQuestions.Edges
	if len(edges) == 0 {
		return "", "", fmt.Errorf("no profile_questions edges in response")
	}
	sess = edges[0].Session
	if sess == "" {
		return "", "", fmt.Errorf("empty session token in response")
	}
	// Extract first inference page ID if available.
	infEdges := edges[0].Node.Inferences.Edges
	if len(infEdges) > 0 {
		inferenceID = infEdges[0].Node.Page.ID
	}
	return sess, inferenceID, nil
}

func callRelayMutation(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, sessID, friendlyName, docID string, vars map[string]interface{}, withAnalytics bool) error {
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, friendlyName, docID, string(varsJSON), sessID, withAnalytics)
	headers := relayHeaders(friendlyName, machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)
	_, err := doPost(ctx, cl, targetURL, body, headers)
	return err
}

// callSaveMutation calls the ProfileWizardProfileFieldSaveMutation and returns the raw response.
func callSaveMutation(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, sessID string, vars map[string]interface{}) (string, error) {
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "ProfileWizardProfileFieldSaveMutation", docSaveMutation, string(varsJSON), sessID, true)
	headers := relayHeaders("ProfileWizardProfileFieldSaveMutation", machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)
	return doPost(ctx, cl, targetURL, body, headers)
}

// saveSchoolDirect saves a high school using ProfileEditEducationExperienceSaveMutation.
// Bypasses the NUX wizard — no session token required.
func saveSchoolDirect(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, schoolID, schoolName string, mutID int, notify func(string)) error {
	notify(fmt.Sprintf("[AddInfo] Saving school direct: %s (id=%s)", schoolName, schoolID))
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"client_mutation_id":      fmt.Sprintf("%d", mutID),
			"actor_id":               uid,
			"concentration_id":       nil,
			"concentration_name":     nil,
			"experience_id":          nil,
			"has_graduated":          false,
			"life_event_publish_type": "SUPPRESS_ALL",
			"privacy":                privacyEveryone(),
			"school_id":              schoolID,
			"school_name":            schoolName,
			"school_type":            "hs",
			"start":                  map[string]interface{}{},
			"end":                    map[string]interface{}{},
			"ref":                    "react_native_form",
			"mutation_surface":       "PROFILE",
			"session_id":             newUUID(),
		},
	}
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "ProfileEditEducationExperienceSaveMutation", docSchoolSave, string(varsJSON), "", false)
	headers := relayHeaders("ProfileEditEducationExperienceSaveMutation", machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)

	resp, err := doPost(ctx, cl, targetURL, body, headers)
	if err != nil {
		return err
	}
	if strings.Contains(resp, `"errors"`) {
		snippet := resp
		if len(snippet) > 250 {
			snippet = snippet[:250]
		}
		return fmt.Errorf("FB error (school): %s", snippet)
	}
	notify(fmt.Sprintf("[AddInfo] Save OK: school → resp: %s", truncate(resp, 80)))
	return nil
}

// saveWorkDirect saves a work experience using ProfileEditWorkViewMutation.
// Bypasses the NUX wizard — no session token required.
func saveWorkDirect(ctx context.Context, cl tls_client.HttpClient, token, uid, locale, machineID, deviceID, workID, workName string, mutID int, notify func(string)) error {
	notify(fmt.Sprintf("[AddInfo] Saving work direct: %s (id=%s)", workName, workID))
	now := time.Now()
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"client_mutation_id":      fmt.Sprintf("%d", mutID),
			"actor_id":               uid,
			"employer_id":            workID,
			"employer_name":          workName,
			"start_date":             map[string]interface{}{"year": now.Year(), "month": int(now.Month()), "day": now.Day()},
			"end_date":               map[string]interface{}{"year": nil, "month": nil, "day": nil},
			"is_current":             true,
			"privacy":                privacyEveryone(),
			"mutation_surface":       "PROFILE",
			"session_id":             newUUID(),
			"life_event_publish_type": "SUPPRESS_ALL",
		},
	}
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "ProfileEditWorkViewMutation", docWorkSave, string(varsJSON), "", false)
	headers := relayHeaders("ProfileEditWorkViewMutation", machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)

	resp, err := doPost(ctx, cl, targetURL, body, headers)
	if err != nil {
		return err
	}
	if strings.Contains(resp, `"errors"`) {
		snippet := resp
		if len(snippet) > 250 {
			snippet = snippet[:250]
		}
		return fmt.Errorf("FB error (work): %s", snippet)
	}
	notify(fmt.Sprintf("[AddInfo] Save OK: work → resp: %s", truncate(resp, 80)))
	return nil
}

func buildBody(token, friendlyName, docID, varsJSON, sessID string, withAnalytics bool) string {
	var sb strings.Builder
	sb.WriteString("access_token=")
	sb.WriteString(url.QueryEscape(token))
	sb.WriteString("&fb_api_caller_class=RelayModern")
	sb.WriteString("&fb_api_req_friendly_name=")
	sb.WriteString(url.QueryEscape(friendlyName))
	sb.WriteString("&server_timestamps=true")
	sb.WriteString("&variables=")
	sb.WriteString(url.QueryEscape(varsJSON))
	sb.WriteString("&doc_id=")
	sb.WriteString(docID)
	if withAnalytics && sessID != "" {
		tags, _ := json.Marshal([]string{
			"session_id:" + sessID,
			"nav_attribution_id=",
		})
		sb.WriteString("&fb_api_analytics_tags=")
		sb.WriteString(url.QueryEscape(string(tags)))
	}
	return sb.String()
}

func privacyEveryone() map[string]interface{} {
	return map[string]interface{}{
		"allow":               []interface{}{},
		"base_state":          "EVERYONE",
		"deny":                []interface{}{},
		"tag_expansion_state": "UNSPECIFIED",
	}
}

// searchCity calls AddressTypeaheadGraphQLSearchSourceQuery to find a city/town Page ID.
// This is the correct API for HOMETOWN — inference is always empty for that field.
// Tries progressively shorter queries if no result is found.
func searchCity(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, cityName string) (string, string, error) {
	queries := buildFallbackQueries(cityName)
	for _, q := range queries {
		id, name, err := doCitySearch(ctx, cl, token, locale, machineID, deviceID, q)
		if err != nil {
			return "", "", err
		}
		if id != "" {
			return id, name, nil
		}
	}
	return "", "", nil
}

func doCitySearch(ctx context.Context, cl tls_client.HttpClient, token, locale, machineID, deviceID, query string) (string, string, error) {
	vars := map[string]interface{}{
		"allowlistedCountries":      nil,
		"locationTypes":             []interface{}{},
		"maxResults":                10,
		"queryString":               query,
		"regulatedCategories":       nil,
		"useGeoLocationSearchQuery": false,
		"queryParams": map[string]interface{}{
			"query": query,
			"viewer_coordinates": map[string]interface{}{
				"latitude":  nil,
				"longitude": nil,
			},
			"provider":                    "here_thrift",
			"search_type":                 "city_typeahead",
			"integration_strategy":        "string_match",
			"result_ordering":             "interleave",
			"caller":                      "profile_about",
			"country_filter":              nil,
			"page_category":               []string{"city", "subcity"},
			"radius":                      nil,
			"geocode_fallback":            false,
			"geo_proximity_search_weight": nil,
		},
	}
	varsJSON, _ := json.Marshal(vars)
	body := buildBody(token, "AddressTypeaheadGraphQLSearchSourceQuery", docCitySearch, string(varsJSON), "", false)
	headers := relayHeaders("AddressTypeaheadGraphQLSearchSourceQuery", machineID, deviceID)
	targetURL := graphURL + "?locale=" + url.QueryEscape(locale)

	respStr, err := doPost(ctx, cl, targetURL, body, headers)
	if err != nil {
		return "", "", fmt.Errorf("city search HTTP: %w", err)
	}
	return parseCitySearchResult(respStr)
}

type citySearchResp struct {
	Data struct {
		CityStreetSearch struct {
			StreetResults struct {
				Edges []struct {
					Node struct {
						Title string `json:"title"`
						Page  struct {
							ID string `json:"id"`
						} `json:"page"`
						City struct {
							ID string `json:"id"`
						} `json:"city"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"street_results"`
		} `json:"city_street_search"`
	} `json:"data"`
}

func parseCitySearchResult(respStr string) (id, name string, err error) {
	var r citySearchResp
	if jsonErr := json.Unmarshal([]byte(respStr), &r); jsonErr != nil {
		snippet := respStr
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		return "", "", fmt.Errorf("parse city search: %w | resp: %s", jsonErr, snippet)
	}
	edges := r.Data.CityStreetSearch.StreetResults.Edges
	if len(edges) == 0 {
		return "", "", nil
	}
	node := edges[0].Node
	id = node.Page.ID
	if id == "" {
		id = node.City.ID
	}
	name = node.Title
	return id, name, nil
}
