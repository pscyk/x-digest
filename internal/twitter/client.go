package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// bearerToken is X's PUBLIC web-client bearer token, identical for every user.
	// Embedded in x.com's JavaScript bundle — not a secret.
	bearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs=1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	homeTimelineQueryID       = "Y37d_0I85ytclAJzNXoUYA"
	homeLatestTimelineQueryID = "PltnwqQPKVs8-mXP-h4lZQ"
	userByScreenNameQueryID   = "IGgvgiOx4QZndDHuD3x9TQ"
	userTweetsQueryID         = "ItymOSM-DiyKKK0lF2YWwA"
	bookmarksQueryID          = "0JDkRvhJCNBZKPqDxd1dgQ"

	graphqlBaseURL = "https://x.com/i/api/graphql"
	apiTimeout     = 30 * time.Second
	maxBodyBytes   = 10 * 1024 * 1024
)

var features = map[string]bool{
	"rweb_tipjar_consumption_enabled":                                         true,
	"responsive_web_graphql_exclude_directive_enabled":                        true,
	"verified_phone_label_enabled":                                            false,
	"creator_subscriptions_tweet_preview_api_enabled":                         true,
	"responsive_web_graphql_timeline_navigation_enabled":                      true,
	"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
	"communities_web_enable_tweet_community_results_fetch":                    true,
	"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
	"articles_preview_enabled":                                                true,
	"responsive_web_edit_tweet_api_enabled":                                   true,
	"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
	"view_counts_everywhere_api_enabled":                                      true,
	"longform_notetweets_consumption_enabled":                                 true,
	"responsive_web_twitter_article_tweet_consumption_enabled":                true,
	"tweet_awards_web_tipping_enabled":                                        false,
	"creator_subscriptions_quote_tweet_preview_enabled":                       false,
	"freedom_of_speech_not_reach_fetch_enabled":                               true,
	"standardized_nudges_misinfo":                                             true,
	"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
	"rweb_video_timestamps_enabled":                                           true,
	"longform_notetweets_rich_text_read_enabled":                              true,
	"longform_notetweets_inline_media_enabled":                                true,
	"responsive_web_enhance_cards_enabled":                                    false,
}

// Client talks to X's internal GraphQL API using browser cookies.
type Client struct {
	authToken string
	ct0       string
	http      *http.Client
}

// NewClient creates a Client from the selected browser's cookies.
func NewClient(authToken, ct0 string) *Client {
	return &Client{
		authToken: authToken,
		ct0:       ct0,
		http: &http.Client{Timeout: apiTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Never forward session headers to a redirect target.
			return http.ErrUseLastResponse
		}},
	}
}

// FetchTimeline fetches the authenticated user's home timeline.
// timelineType is "following" or "foryou". queryIDOverride replaces the default query ID.
func (c *Client) FetchTimeline(timelineType string, count int, queryIDOverride string) ([]Tweet, error) {
	var queryID, endpoint string
	switch timelineType {
	case "foryou":
		queryID = homeTimelineQueryID
		endpoint = "HomeTimeline"
	default:
		queryID = homeLatestTimelineQueryID
		endpoint = "HomeLatestTimeline"
	}
	if queryIDOverride != "" {
		queryID = queryIDOverride
	}

	variables := map[string]any{
		"count":                  count * 3,
		"includePromotedContent": true,
		"latestControlAvailable": true,
		"requestContext":         "launch",
		"withCommunity":          true,
		"seenTweetIds":           make([]string, 0),
	}

	body, err := c.graphqlGET(queryID, endpoint, variables)
	if err != nil {
		return nil, err
	}

	var resp timelineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing timeline JSON: %w", err)
	}
	return parseInstructions(resp.Data.Home.HomeTimelineUrt.Instructions)
}

// FetchUserTweets resolves a screen name and fetches their recent tweets.
func (c *Client) FetchUserTweets(screenName string, count int) ([]Tweet, error) {
	userID, err := c.resolveScreenName(screenName)
	if err != nil {
		return nil, err
	}

	variables := map[string]any{
		"userId":                                 userID,
		"count":                                  count * 2,
		"includePromotedContent":                 true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	body, err := c.graphqlGET(userTweetsQueryID, "UserTweets", variables)
	if err != nil {
		return nil, err
	}

	var resp userTweetsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing user tweets JSON: %w", err)
	}
	return parseInstructions(resp.Data.User.Result.Timeline.Timeline.Instructions)
}

// FetchBookmarks searches the authenticated user's bookmarks.
// query filters bookmarks; empty string searches broadly.
func (c *Client) FetchBookmarks(count int, query string) ([]Tweet, error) {
	if query == "" {
		query = "a" // broad match — no "list all" endpoint in X's public bundle
	}
	variables := map[string]any{
		"rawQuery":               query,
		"count":                  count * 2,
		"includePromotedContent": false,
	}

	body, err := c.graphqlGET(bookmarksQueryID, "BookmarkSearchTimeline", variables)
	if err != nil {
		return nil, err
	}

	// Discover response path dynamically.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing bookmarks JSON: %w", err)
	}

	instructions, err := extractInstructionsFromUnknownPath(raw)
	if err != nil {
		return nil, fmt.Errorf("bookmarks: %w", err)
	}

	return parseInstructions(instructions)
}

// SearchTweets searches X for tweets matching the query using the adaptive search API.
func (c *Client) resolveScreenName(screenName string) (string, error) {
	variables := map[string]any{
		"screen_name":              screenName,
		"withSafetyModeUserFields": true,
	}

	body, err := c.graphqlGET(userByScreenNameQueryID, "UserByScreenName", variables)
	if err != nil {
		return "", fmt.Errorf("looking up @%s: %w", screenName, err)
	}

	var resp userByScreenNameResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing user lookup: %w", err)
	}

	uid := resp.Data.User.Result.RestID
	if uid == "" {
		return "", fmt.Errorf("user @%s not found", screenName)
	}
	return uid, nil
}

func (c *Client) graphqlGETWithToggles(queryID, endpoint string, variables map[string]any, fieldToggles map[string]any) ([]byte, error) {
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("marshalling variables: %w", err)
	}
	featJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("marshalling features: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s/%s", graphqlBaseURL, queryID, endpoint)
	u, err := url.Parse(reqURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	if fieldToggles != nil {
		togglesJSON, _ := json.Marshal(fieldToggles)
		q.Set("fieldToggles", string(togglesJSON))
	}
	u.RawQuery = q.Encode()

	return c.doRequest(u.String())
}

// graphqlGET builds and executes a GET request to X's GraphQL API.
func (c *Client) graphqlGET(queryID, endpoint string, variables map[string]any) ([]byte, error) {
	return c.graphqlGETWithToggles(queryID, endpoint, variables, nil)
}

func (c *Client) doRequest(reqURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("X-Csrf-Token", c.ct0)
	req.Header.Set("Cookie", fmt.Sprintf("auth_token=%s; ct0=%s", c.authToken, c.ct0))
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	switch resp.StatusCode {
	case 200:
		return body, nil
	case 401, 403:
		return nil, fmt.Errorf("authentication failed (HTTP %d) — your X session may have expired, log into x.com in the selected browser and retry", resp.StatusCode)
	case 429:
		return nil, fmt.Errorf("rate limited (HTTP 429) — wait a few minutes and try again")
	default:
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, preview)
	}
}
