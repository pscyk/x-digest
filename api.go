package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

	graphqlBaseURL = "https://x.com/i/api/graphql"

	apiTimeout     = 30 * time.Second
	maxBodyBytes   = 10 * 1024 * 1024 // 10 MB safety cap on response reads
	featureCount   = 22               // number of feature flags, keep in sync with map below
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

func init() {
	assert(len(features) == featureCount, "feature map count out of sync with featureCount const")
}

// fetchTimeline fetches the authenticated user's home timeline.
func fetchTimeline(authToken, ct0, timelineType string, count int, queryIDOverride string) ([]Tweet, error) {
	assert(authToken != "", "authToken must not be empty")
	assert(ct0 != "", "ct0 must not be empty")
	assert(count > 0, "count must be positive")

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
		"count":                  count * 3, // over-request; we filter + sort client-side
		"includePromotedContent": true,
		"latestControlAvailable": true,
		"requestContext":         "launch",
		"withCommunity":          true,
		"seenTweetIds":           make([]string, 0),
	}

	body, err := graphqlGET(queryID, endpoint, variables, authToken, ct0)
	if err != nil {
		return nil, err
	}

	return parseHomeTimeline(body)
}

// fetchUserTweets resolves a screen name and fetches their recent tweets.
func fetchUserTweets(authToken, ct0, screenName string, count int) ([]Tweet, error) {
	assert(screenName != "", "screenName must not be empty")
	assert(count > 0, "count must be positive")

	userID, err := resolveScreenName(authToken, ct0, screenName)
	if err != nil {
		return nil, err
	}

	variables := map[string]any{
		"userId":                 userID,
		"count":                  count * 2,
		"includePromotedContent": true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	body, err := graphqlGET(userTweetsQueryID, "UserTweets", variables, authToken, ct0)
	if err != nil {
		return nil, err
	}

	var resp UserTweetsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing user tweets JSON: %w", err)
	}

	return parseInstructions(resp.Data.User.Result.Timeline.Timeline.Instructions)
}

func resolveScreenName(authToken, ct0, screenName string) (string, error) {
	variables := map[string]any{
		"screen_name":              screenName,
		"withSafetyModeUserFields": true,
	}

	body, err := graphqlGET(userByScreenNameQueryID, "UserByScreenName", variables, authToken, ct0)
	if err != nil {
		return "", fmt.Errorf("looking up @%s: %w", screenName, err)
	}

	var resp UserByScreenNameResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing user lookup: %w", err)
	}

	uid := resp.Data.User.Result.RestID
	if uid == "" {
		return "", fmt.Errorf("user @%s not found", screenName)
	}
	return uid, nil
}

// --- HTTP layer ---

// graphqlGET builds and executes a GET request to X's GraphQL API.
func graphqlGET(queryID, endpoint string, variables map[string]any, authToken, ct0 string) ([]byte, error) {
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("marshalling variables: %w", err)
	}
	featJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("marshalling features: %w", err)
	}

	u, err := url.Parse(fmt.Sprintf("%s/%s/%s", graphqlBaseURL, queryID, endpoint))
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	u.RawQuery = q.Encode()

	return doAPIRequest(u.String(), authToken, ct0)
}

func doAPIRequest(reqURL, authToken, ct0 string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("X-Csrf-Token", ct0)
	req.Header.Set("Cookie", fmt.Sprintf("auth_token=%s; ct0=%s", authToken, ct0))
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Do(req)
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
		return nil, fmt.Errorf("authentication failed (HTTP %d) — your X session may have expired, log into x.com in Firefox and retry", resp.StatusCode)
	case 429:
		return nil, fmt.Errorf("rate limited (HTTP 429) — wait a few minutes and try again")
	default:
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
}

// --- JSON parsing ---

func parseHomeTimeline(body []byte) ([]Tweet, error) {
	var resp TimelineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response JSON: %w", err)
	}
	return parseInstructions(resp.Data.Home.HomeTimelineUrt.Instructions)
}

func parseInstructions(instructions []json.RawMessage) ([]Tweet, error) {
	// Pre-allocate with estimated capacity: ~50 entries per timeline fetch.
	tweets := make([]Tweet, 0, 50)

	for _, rawInstr := range instructions {
		var instr Instruction
		if err := json.Unmarshal(rawInstr, &instr); err != nil {
			continue
		}
		if instr.Type != "TimelineAddEntries" {
			continue
		}

		var addEntries AddEntriesInstruction
		if err := json.Unmarshal(rawInstr, &addEntries); err != nil {
			continue
		}

		for _, entry := range addEntries.Entries {
			if strings.HasPrefix(entry.EntryID, "cursor-") {
				continue
			}

			if strings.HasPrefix(entry.EntryID, "tweet-") {
				if t, ok := extractTweetFromItem(entry.Content); ok {
					tweets = append(tweets, t)
				}
				continue
			}

			if strings.HasPrefix(entry.EntryID, "conversationthread-") || strings.HasPrefix(entry.EntryID, "profile-conversation-") {
				var mod TimelineModuleContent
				if err := json.Unmarshal(entry.Content, &mod); err == nil && mod.Items != nil {
					for _, item := range mod.Items {
						if t, ok := extractFromTweetContent(item.Item.ItemContent); ok {
							tweets = append(tweets, t)
						}
					}
				}
				continue
			}

			// Fallback: try extracting a tweet from any unknown entry type.
			if t, ok := extractTweetFromItem(entry.Content); ok {
				tweets = append(tweets, t)
			}
		}
	}

	return tweets, nil
}

func extractTweetFromItem(raw json.RawMessage) (Tweet, bool) {
	var item TimelineItemContent
	if err := json.Unmarshal(raw, &item); err != nil {
		return Tweet{}, false
	}
	return extractFromTweetContent(item.ItemContent)
}

func extractFromTweetContent(tc TweetContent) (Tweet, bool) {
	if tc.Typename != "TimelineTweet" && tc.Typename != "" {
		return Tweet{}, false
	}
	if tc.TweetResults.Result == nil {
		return Tweet{}, false
	}

	var result TweetResult
	if err := json.Unmarshal(tc.TweetResults.Result, &result); err != nil {
		return Tweet{}, false
	}

	// TweetWithVisibilityResults wraps the real tweet one level deeper.
	if result.Typename == "TweetWithVisibilityResults" && result.Tweet != nil {
		if err := json.Unmarshal(result.Tweet, &result); err != nil {
			return Tweet{}, false
		}
	}

	if result.Typename != "Tweet" && result.Typename != "" {
		return Tweet{}, false
	}

	legacy := result.Legacy
	if legacy.IDStr == "" {
		return Tweet{}, false
	}

	views, _ := strconv.Atoi(result.Views.Count)
	ts, _ := time.Parse("Mon Jan 02 15:04:05 -0700 2006", legacy.CreatedAt)

	mediaType := ""
	hasMedia := false
	mediaList := legacy.ExtendedEntities.Media
	if len(mediaList) == 0 {
		mediaList = legacy.Entities.Media
	}
	if len(mediaList) > 0 {
		hasMedia = true
		mediaType = mediaList[0].Type
	}

	// User info lives in .legacy on home timeline, .core on user timeline.
	authorName := result.Core.UserResults.Result.Legacy.Name
	authorHandle := result.Core.UserResults.Result.Legacy.ScreenName
	if authorName == "" {
		authorName = result.Core.UserResults.Result.UserCore.Name
	}
	if authorHandle == "" {
		authorHandle = result.Core.UserResults.Result.UserCore.ScreenName
	}

	return Tweet{
		ID:           legacy.IDStr,
		Text:         legacy.FullText,
		AuthorName:   authorName,
		AuthorHandle: authorHandle,
		Likes:        legacy.FavoriteCount,
		Retweets:     legacy.RetweetCount,
		Replies:      legacy.ReplyCount,
		Quotes:       legacy.QuoteCount,
		Views:        views,
		Timestamp:    ts,
		HasMedia:     hasMedia,
		MediaType:    mediaType,
		IsVerified:   result.Core.UserResults.Result.IsBlueVerified,
		Engagement:   legacy.FavoriteCount + legacy.RetweetCount,
	}, true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
