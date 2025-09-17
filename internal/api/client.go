// Package api provides the NotebookLM API client.
package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	pb "github.com/tmc/nlm/gen/notebooklm/v1alpha1"
	"github.com/tmc/nlm/internal/batchexecute"
	"github.com/tmc/nlm/internal/beprotojson"
	"github.com/tmc/nlm/internal/rpc"
)

// Time threshold constants for Google Drive sync analysis
const (
	// TenDaysInSeconds represents 10 days in seconds (864000 seconds)
	TenDaysInSeconds = 10 * 24 * 60 * 60
	// OneDayInSeconds represents 1 day in seconds (86400 seconds)
	OneDayInSeconds = 24 * 60 * 60
)

type Notebook = pb.Project
type Note = pb.Source

// Client handles NotebookLM API interactions.
type Client struct {
	rpc *rpc.Client
}

// New creates a new NotebookLM API client.
func New(authToken, cookies string, opts ...batchexecute.Option) *Client {
	return &Client{
		rpc: rpc.New(authToken, cookies, opts...),
	}
}

// Project/Notebook operations

func (c *Client) ListRecentlyViewedProjects() ([]*Notebook, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCListRecentlyViewedProjects,
		Args: []interface{}{nil, 1},
	})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	var response pb.ListRecentlyViewedProjectsResponse
	if err := beprotojson.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return response.Projects, nil
}

func (c *Client) CreateProject(title string, emoji string) (*Notebook, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCCreateProject,
		Args: []interface{}{title, emoji},
	})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	var project pb.Project
	if err := beprotojson.Unmarshal(resp, &project); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &project, nil
}

func (c *Client) GetProject(projectID string) (*Notebook, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGetProject,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	// Debug: Print raw response before unmarshaling
	if c.rpc.Config.Debug {
		fmt.Fprintf(os.Stderr, "=== GetProject Raw Response ===\n")
		fmt.Fprintf(os.Stderr, "Response length: %d bytes\n", len(resp))
		previewLen := 500
		if len(resp) < previewLen {
			previewLen = len(resp)
		}
		fmt.Fprintf(os.Stderr, "Response preview: %s\n", string(resp[:previewLen]))
		fmt.Fprintf(os.Stderr, "================================\n")
	}

	// Sources nesting issue is now fixed in beprotojson package

	var project pb.Project
	if err := beprotojson.Unmarshal(resp, &project); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Debug: Print parsed project after unmarshaling
	if c.rpc.Config.Debug {
		fmt.Fprintf(os.Stderr, "=== GetProject Parsed Result ===\n")
		fmt.Fprintf(os.Stderr, "Project ID: '%s'\n", project.ProjectId)
		fmt.Fprintf(os.Stderr, "Project Title: '%s'\n", project.Title)
		fmt.Fprintf(os.Stderr, "Project Emoji: '%s'\n", project.Emoji)
		fmt.Fprintf(os.Stderr, "Sources count: %d\n", len(project.Sources))
		if len(project.Sources) > 0 {
			fmt.Fprintf(os.Stderr, "First source: %+v\n", project.Sources[0])
		}
		fmt.Fprintf(os.Stderr, "=================================\n")
	}

	return &project, nil
}

func (c *Client) DeleteProjects(projectIDs []string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCDeleteProjects,
		Args: []interface{}{projectIDs},
	})
	if err != nil {
		return fmt.Errorf("delete projects: %w", err)
	}
	return nil
}

func (c *Client) MutateProject(projectID string, updates *pb.Project) (*Notebook, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCMutateProject,
		Args:       []interface{}{projectID, updates},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("mutate project: %w", err)
	}

	var project pb.Project
	if err := beprotojson.Unmarshal(resp, &project); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &project, nil
}

func (c *Client) RemoveRecentlyViewedProject(projectID string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCRemoveRecentlyViewed,
		Args: []interface{}{projectID},
	})
	return err
}

// Source operations

/*
func (c *Client) AddSources(projectID string, sources []*pb.Source) ([]*pb.Source, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCAddSources,
		Args:       []interface{}{projectID, sources},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("add sources: %w", err)
	}

	var result []*pb.Source
	if err := beprotojson.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}
*/

func (c *Client) DeleteSources(projectID string, sourceIDs []string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCDeleteSources,
		Args: []interface{}{
			[][][]string{{sourceIDs}},
		},
		NotebookID: projectID,
	})
	return err
}

func (c *Client) MutateSource(sourceID string, updates *pb.Source) (*pb.Source, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCMutateSource,
		Args: []interface{}{sourceID, updates},
	})
	if err != nil {
		return nil, fmt.Errorf("mutate source: %w", err)
	}

	var source pb.Source
	if err := beprotojson.Unmarshal(resp, &source); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &source, nil
}

func (c *Client) RefreshSource(sourceID string) (*pb.Source, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:   rpc.RPCRefreshSource,
		Args: []interface{}{sourceID},
	})
	if err != nil {
		return nil, fmt.Errorf("refresh source: %w", err)
	}

	var source pb.Source
	if err := beprotojson.Unmarshal(resp, &source); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &source, nil
}

func (c *Client) LoadSource(sourceID string) (*pb.Source, error) {
	// Use DoWithFullResponse to get both parsed data and raw response for debugging
	fullResp, err := c.rpc.DoWithFullResponse(rpc.Call{
		ID:   rpc.RPCLoadSource,
		Args: []interface{}{sourceID},
	})
	if err != nil {
		return nil, fmt.Errorf("load source: %w", err)
	}

	if c.rpc.Config.Debug {
		fmt.Printf("=== LoadSource Raw Response for %s ===\n", sourceID)
		fmt.Printf("RawArray length: %d\n", len(fullResp.RawArray))
		if len(fullResp.RawArray) > 0 {
			for i, item := range fullResp.RawArray {
				fmt.Printf("  [%d]: %v (type: %T)\n", i, item, item)
			}
		}
		fmt.Printf("Raw JSON Data: %s\n", string(fullResp.Data))
		fmt.Printf("==========================================\n")
	}

	var source pb.Source
	if err := beprotojson.Unmarshal(fullResp.Data, &source); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if c.rpc.Config.Debug {
		fmt.Printf("=== Parsed Source Metadata ===\n")
		if source.SourceId != nil {
			fmt.Printf("Source ID: %s\n", source.SourceId.SourceId)
		}
		fmt.Printf("Title: %s\n", source.Title)
		if source.Metadata != nil {
			fmt.Printf("Source Type: %s\n", source.Metadata.SourceType.String())
			if gdMeta := source.Metadata.GetGoogleDocs(); gdMeta != nil {
				fmt.Printf("Google Docs Document ID: %s\n", gdMeta.DocumentId)
			}
		}
		if source.Settings != nil {
			fmt.Printf("Source Status: %s\n", source.Settings.Status.String())
		}
		fmt.Printf("==============================\n")
	}

	return &source, nil
}

// SourceFreshnessResult represents the result of a source freshness check
type SourceFreshnessResult struct {
	SourceID string
	Status   pb.SourceSettings_SourceStatus
	Message  string
}

func (c *Client) CheckSourceFreshness(sourceID string) (*SourceFreshnessResult, error) {
	result := &SourceFreshnessResult{
		SourceID: sourceID,
	}

	// INSIGHT: CheckSourceFreshness API consistently returns [3] for all sources,
	// regardless of their sync status. This suggests it's not the right API for
	// determining Google Drive sync status.
	//
	// Instead, we'll analyze the source metadata from the project listing
	// to determine sync status based on the presence and structure of Google Drive metadata.

	return c.checkSourceSyncStatus(sourceID, result)
}

func (c *Client) checkSourceSyncStatus(sourceID string, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	// Get raw project data to analyze source sync status directly from API response
	// This bypasses protobuf parsing issues and analyzes the raw JSON structure

	resp, err := c.rpc.DoWithFullResponse(rpc.Call{
		ID:   rpc.RPCListRecentlyViewedProjects,
		Args: []interface{}{nil, 1},
	})
	if err != nil {
		result.Status = pb.SourceSettings_SOURCE_STATUS_ERROR
		result.Message = fmt.Sprintf("Failed to get projects data: %v", err)
		return result, nil
	}

	// Parse the raw response to find our source
	var responseData []interface{}
	if err := json.Unmarshal(resp.Data, &responseData); err != nil {
		result.Status = pb.SourceSettings_SOURCE_STATUS_ERROR
		result.Message = fmt.Sprintf("Failed to parse projects response: %v", err)
		return result, nil
	}

	// Search through projects in the raw response
	if len(responseData) > 0 {
		if projects, ok := responseData[0].([]interface{}); ok {
			for _, projectData := range projects {
				if project, ok := projectData.([]interface{}); ok && len(project) > 1 {
					// project[0] = title, project[1] = sources array
					if sourcesData, ok := project[1].([]interface{}); ok {
						for _, sourceData := range sourcesData {
							if sourceArr, ok := sourceData.([]interface{}); ok && len(sourceArr) > 2 {
								// sourceArr[0] = [sourceID], sourceArr[1] = title, sourceArr[2] = metadata
								if sourceIDArr, ok := sourceArr[0].([]interface{}); ok && len(sourceIDArr) > 0 {
									if sourceIDStr, ok := sourceIDArr[0].(string); ok && sourceIDStr == sourceID {
										// Found our source - analyze its metadata structure
										return c.analyzeRawSourceStructure(sourceArr, result)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	result.Status = pb.SourceSettings_SOURCE_STATUS_ERROR
	result.Message = "Source not found in any project"
	return result, nil
}

func (c *Client) extractSourceTitle(sourceArr []interface{}) string {
	if title, ok := sourceArr[1].(string); ok {
		return title
	}
	return "Unknown Source"
}

func (c *Client) debugSourceStructure(sourceTitle string, sourceArr []interface{}) {
	if !c.rpc.Config.Debug {
		return
	}
	fmt.Printf("=== Detailed Source Analysis ===\n")
	fmt.Printf("Source Title: %s\n", sourceTitle)
	fmt.Printf("Full source array length: %d\n", len(sourceArr))
	for i, elem := range sourceArr {
		fmt.Printf("Position [%d]: %T = %+v\n", i, elem, elem)
	}
	fmt.Printf("==============================\n")
}

func (c *Client) debugMetadata(metadataArr []interface{}) {
	if !c.rpc.Config.Debug {
		return
	}
	fmt.Printf("Metadata array length: %d\n", len(metadataArr))
	for i, elem := range metadataArr {
		fmt.Printf("Metadata [%d]: %T = %+v\n", i, elem, elem)
	}
}

func (c *Client) isGoogleDriveSource(metadataArr []interface{}) bool {
	if metadataArr[0] == nil {
		return false
	}
	googleDriveInfo, ok := metadataArr[0].([]interface{})
	return ok && len(googleDriveInfo) >= 1
}

func (c *Client) setRegularSourceStatus(result *SourceFreshnessResult, sourceTitle string) *SourceFreshnessResult {
	result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
	if sourceTitle != "Unknown Source" {
		result.Message = fmt.Sprintf("Source (%s) is functioning normally", sourceTitle)
	} else {
		result.Message = "Source is functioning normally"
	}
	return result
}

func (c *Client) analyzeGoogleDriveSync(metadataArr []interface{}, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	if c.rpc.Config.Debug {
		fmt.Printf("Google Drive source detected. Metadata array length: %d\n", len(metadataArr))
	}

	switch len(metadataArr) {
	case 7:
		return c.analyzeLength7Metadata(metadataArr, result)
	case 6:
		return c.analyzeLength6Metadata(metadataArr, result)
	case 5:
		return c.analyzeLength5Metadata(metadataArr, result)
	default:
		result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
		result.Message = "Google Drive source is properly synchronized"
		return result, nil
	}
}

func (c *Client) analyzeLength7Metadata(metadataArr []interface{}, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	if len(metadataArr) > 5 && metadataArr[5] == nil {
		result.Status = pb.SourceSettings_SOURCE_STATUS_DISABLED
		result.Message = "Google Drive source needs synchronization (クリックして Google ドライブと同期)"
	} else {
		result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
		result.Message = "Google Drive source is properly synchronized"
	}
	return result, nil
}

func (c *Client) analyzeLength6Metadata(metadataArr []interface{}, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	if len(metadataArr) <= 5 {
		result.Status = pb.SourceSettings_SOURCE_STATUS_DISABLED
		result.Message = "Google Drive source needs synchronization (クリックして Google ドライブと同期)"
		return result, nil
	}

	if c.rpc.Config.Debug {
		fmt.Printf("Length 6 source - Position [5]: %+v\n", metadataArr[5])
	}

	if syncFlag, ok := metadataArr[5].(float64); ok && syncFlag == 1 {
		result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
		result.Message = "Google Drive source is properly synchronized"
	} else {
		result.Status = pb.SourceSettings_SOURCE_STATUS_DISABLED
		result.Message = "Google Drive source needs synchronization (クリックして Google ドライブと同期)"
	}
	return result, nil
}

func (c *Client) analyzeLength5Metadata(metadataArr []interface{}, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	if c.rpc.Config.Debug {
		fmt.Printf("Length 5 source - Position [4]: %+v\n", metadataArr[4])
	}

	if syncFlag, ok := metadataArr[4].(float64); ok && syncFlag == 1 {
		return c.analyzeTimestampDifference(metadataArr, result, true)
	}
	return c.analyzeTimestampDifference(metadataArr, result, false)
}

func (c *Client) analyzeTimestampDifference(metadataArr []interface{}, result *SourceFreshnessResult, hasPositionFlag bool) (*SourceFreshnessResult, error) {
	lastUpdate, creation := c.extractTimestamps(metadataArr)

	if c.rpc.Config.Debug {
		if hasPositionFlag {
			fmt.Printf("Length 5 source with position [4] = 1 - Creation: %d, LastUpdate: %d, Diff: %d\n", creation, lastUpdate, lastUpdate-creation)
		} else {
			fmt.Printf("Length 5 source - Creation: %d, LastUpdate: %d\n", creation, lastUpdate)
		}
	}

	if hasPositionFlag {
		// For sources with position [4] = 1, smaller time differences need sync
		if lastUpdate > creation && (lastUpdate-creation) < TenDaysInSeconds {
			result.Status = pb.SourceSettings_SOURCE_STATUS_DISABLED
			result.Message = "Google Drive source needs synchronization (クリックして Google ドライブと同期)"
		} else {
			result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
			result.Message = "Google Drive source is properly synchronized"
		}
	} else {
		// Counter-intuitive logic based on user feedback
		if lastUpdate > creation && (lastUpdate-creation) > OneDayInSeconds {
			result.Status = pb.SourceSettings_SOURCE_STATUS_DISABLED
			result.Message = "Google Drive source needs synchronization (クリックして Google ドライブと同期)"
		} else {
			result.Status = pb.SourceSettings_SOURCE_STATUS_ENABLED
			result.Message = "Google Drive source is properly synchronized"
		}
	}
	return result, nil
}

func (c *Client) extractTimestamps(metadataArr []interface{}) (lastUpdate, creation int64) {
	// Extract timestamps from position [3] and [2]
	if timestampArr, ok := metadataArr[3].([]interface{}); ok && len(timestampArr) >= 2 {
		if ts, ok := timestampArr[1].([]interface{}); ok && len(ts) >= 1 {
			if val, ok := ts[0].(float64); ok {
				lastUpdate = int64(val)
			}
		}
	}
	if timestampArr, ok := metadataArr[2].([]interface{}); ok && len(timestampArr) >= 1 {
		if val, ok := timestampArr[0].(float64); ok {
			creation = int64(val)
		}
	}
	return
}

func (c *Client) analyzeRawSourceStructure(sourceArr []interface{}, result *SourceFreshnessResult) (*SourceFreshnessResult, error) {
	if len(sourceArr) < 4 {
		result.Status = pb.SourceSettings_SOURCE_STATUS_ERROR
		result.Message = "Invalid source structure"
		return result, nil
	}

	sourceTitle := c.extractSourceTitle(sourceArr)
	c.debugSourceStructure(sourceTitle, sourceArr)

	metadataArr, ok := sourceArr[2].([]interface{})
	if !ok || len(metadataArr) == 0 {
		return c.setRegularSourceStatus(result, sourceTitle), nil
	}

	c.debugMetadata(metadataArr)

	if !c.isGoogleDriveSource(metadataArr) {
		return c.setRegularSourceStatus(result, sourceTitle), nil
	}

	finalResult, err := c.analyzeGoogleDriveSync(metadataArr, result)
	if err != nil {
		return finalResult, err
	}

	// Add final debug output
	if c.rpc.Config.Debug {
		fmt.Printf("=== Final Analysis ===\n")
		fmt.Printf("Source Title: %s\n", sourceTitle)
		fmt.Printf("Final Status: %s\n", finalResult.Status.String())
		fmt.Printf("Final Message: %s\n", finalResult.Message)
		fmt.Printf("====================\n")
	}

	return finalResult, nil
}

func (c *Client) getStatusMessage(status pb.SourceSettings_SourceStatus) string {
	switch status {
	case pb.SourceSettings_SOURCE_STATUS_ENABLED:
		return "Source is up to date and available"
	case pb.SourceSettings_SOURCE_STATUS_DISABLED:
		return "Source is disabled"
	case pb.SourceSettings_SOURCE_STATUS_ERROR:
		return "Source has errors and may need to be refreshed"
	default:
		return "Source status unknown"
	}
}

func (c *Client) ActOnSources(projectID string, action string, sourceIDs []string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCActOnSources,
		Args:       []interface{}{projectID, action, sourceIDs},
		NotebookID: projectID,
	})
	return err
}

// Source upload utility methods

func (c *Client) AddSourceFromReader(projectID string, r io.Reader, filename string) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read content: %w", err)
	}

	contentType := http.DetectContentType(content)

	if strings.HasPrefix(contentType, "text/") {
		return c.AddSourceFromText(projectID, string(content), filename)
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	return c.AddSourceFromBase64(projectID, encoded, filename, contentType)
}

func (c *Client) AddSourceFromText(projectID string, content, title string) (string, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCAddSources,
		NotebookID: projectID,
		Args: []interface{}{
			[]interface{}{
				[]interface{}{
					nil,
					[]string{
						title,
						content,
					},
					nil,
					2, // text source type
				},
			},
			projectID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("add text source: %w", err)
	}

	sourceID, err := extractSourceID(resp)
	if err != nil {
		return "", fmt.Errorf("extract source ID: %w", err)
	}
	return sourceID, nil
}

func (c *Client) AddSourceFromBase64(projectID string, content, filename, contentType string) (string, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCAddSources,
		NotebookID: projectID,
		Args: []interface{}{
			[]interface{}{
				[]interface{}{
					content,
					filename,
					contentType,
					"base64",
				},
			},
			projectID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("add binary source: %w", err)
	}

	sourceID, err := extractSourceID(resp)
	if err != nil {
		fmt.Fprintln(os.Stderr, resp)
		spew.Dump(resp)
		return "", fmt.Errorf("extract source ID: %w", err)
	}
	return sourceID, nil
}

func (c *Client) AddSourceFromFile(projectID string, filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return c.AddSourceFromReader(projectID, f, filepath)
}

func (c *Client) AddSourceFromURL(projectID string, url string) (string, error) {
	// Check if it's a YouTube URL first
	if isYouTubeURL(url) {
		videoID, err := extractYouTubeVideoID(url)
		if err != nil {
			return "", fmt.Errorf("invalid YouTube URL: %w", err)
		}
		// Use dedicated YouTube method
		return c.AddYouTubeSource(projectID, videoID)
	}

	// Regular URL handling
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCAddSources,
		NotebookID: projectID,
		Args: []interface{}{
			[]interface{}{
				[]interface{}{
					nil,
					nil,
					[]string{url},
				},
			},
			projectID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("add source from URL: %w", err)
	}

	sourceID, err := extractSourceID(resp)
	if err != nil {
		return "", fmt.Errorf("extract source ID: %w", err)
	}
	return sourceID, nil
}

func (c *Client) AddYouTubeSource(projectID, videoID string) (string, error) {
	if c.rpc.Config.Debug {
		fmt.Printf("=== AddYouTubeSource ===\n")
		fmt.Printf("Project ID: %s\n", projectID)
		fmt.Printf("Video ID: %s\n", videoID)
	}

	// Modified payload structure for YouTube
	payload := []interface{}{
		[]interface{}{
			[]interface{}{
				nil,                                     // content
				nil,                                     // title
				videoID,                                 // video ID (not in array)
				nil,                                     // unused
				pb.SourceType_SOURCE_TYPE_YOUTUBE_VIDEO, // source type
			},
		},
		projectID,
	}

	if c.rpc.Config.Debug {
		fmt.Printf("\nPayload Structure:\n")
		spew.Dump(payload)
	}

	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCAddSources,
		NotebookID: projectID,
		Args:       payload,
	})
	if err != nil {
		return "", fmt.Errorf("add YouTube source: %w", err)
	}

	if c.rpc.Config.Debug {
		fmt.Printf("\nRaw Response:\n%s\n", string(resp))
	}

	if len(resp) == 0 {
		return "", fmt.Errorf("empty response from server (check debug output for request details)")
	}

	sourceID, err := extractSourceID(resp)
	if err != nil {
		return "", fmt.Errorf("extract source ID: %w", err)
	}
	return sourceID, nil
}

// Helper function to extract source ID with better error handling
func extractSourceID(resp json.RawMessage) (string, error) {
	if len(resp) == 0 {
		return "", fmt.Errorf("empty response")
	}

	var data []interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("parse response JSON: %w", err)
	}

	// Try different response formats
	// Format 1: [[[["id",...]]]]
	// Format 2: [[["id",...]]]
	// Format 3: [["id",...]]
	for _, format := range []func([]interface{}) (string, bool){
		// Format 1
		func(d []interface{}) (string, bool) {
			if len(d) > 0 {
				if d0, ok := d[0].([]interface{}); ok && len(d0) > 0 {
					if d1, ok := d0[0].([]interface{}); ok && len(d1) > 0 {
						if d2, ok := d1[0].([]interface{}); ok && len(d2) > 0 {
							if id, ok := d2[0].(string); ok {
								return id, true
							}
						}
					}
				}
			}
			return "", false
		},
		// Format 2
		func(d []interface{}) (string, bool) {
			if len(d) > 0 {
				if d0, ok := d[0].([]interface{}); ok && len(d0) > 0 {
					if d1, ok := d0[0].([]interface{}); ok && len(d1) > 0 {
						if id, ok := d1[0].(string); ok {
							return id, true
						}
					}
				}
			}
			return "", false
		},
		// Format 3
		func(d []interface{}) (string, bool) {
			if len(d) > 0 {
				if d0, ok := d[0].([]interface{}); ok && len(d0) > 0 {
					if id, ok := d0[0].(string); ok {
						return id, true
					}
				}
			}
			return "", false
		},
	} {
		if id, ok := format(data); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("could not find source ID in response structure: %v", data)
}

// Note operations

func (c *Client) CreateNote(projectID string, title string, initialContent string) (*Note, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCCreateNote,
		Args: []interface{}{
			projectID,
			initialContent,
			[]int{1}, // note type
			nil,
			title,
		},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	var note Note
	if err := beprotojson.Unmarshal(resp, &note); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &note, nil
}

func (c *Client) MutateNote(projectID string, noteID string, content string, title string) (*Note, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCMutateNote,
		Args: []interface{}{
			projectID,
			noteID,
			[][][]interface{}{{
				{content, title, []interface{}{}},
			}},
		},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("mutate note: %w", err)
	}

	var note Note
	if err := beprotojson.Unmarshal(resp, &note); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &note, nil
}

func (c *Client) DeleteNotes(projectID string, noteIDs []string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCDeleteNotes,
		Args: []interface{}{
			[][][]string{{noteIDs}},
		},
		NotebookID: projectID,
	})
	return err
}

func (c *Client) GetNotes(projectID string) ([]*Note, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGetNotes,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("get notes: %w", err)
	}

	var response pb.GetNotesResponse
	if err := beprotojson.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return response.Notes, nil
}

// Audio operations

func (c *Client) CreateAudioOverview(projectID string, instructions string) (*AudioOverviewResult, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID required")
	}
	if instructions == "" {
		return nil, fmt.Errorf("instructions required")
	}

	resp, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCCreateAudioOverview,
		Args: []interface{}{
			projectID,
			0,
			[]string{
				instructions,
			},
		},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("create audio overview: %w", err)
	}

	var data []interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}

	result := &AudioOverviewResult{
		ProjectID: projectID,
	}

	// Handle empty or nil response
	if len(data) == 0 {
		return result, nil
	}

	// Parse the wrb.fr response format
	// Format: [null,null,[3,"<base64-audio>","<id>","<title>",null,true],null,[false]]
	if len(data) > 2 {
		audioData, ok := data[2].([]interface{})
		if !ok || len(audioData) < 4 {
			// Creation might be in progress, return result without error
			return result, nil
		}

		// Extract audio data (index 1)
		if audioBase64, ok := audioData[1].(string); ok {
			result.AudioData = audioBase64
		}

		// Extract ID (index 2)
		if id, ok := audioData[2].(string); ok {
			result.AudioID = id
		}

		// Extract title (index 3)
		if title, ok := audioData[3].(string); ok {
			result.Title = title
		}

		// Extract ready status (index 5)
		if len(audioData) > 5 {
			if ready, ok := audioData[5].(bool); ok {
				result.IsReady = ready
			}
		}
	}

	return result, nil
}

func (c *Client) GetAudioOverview(projectID string) (*AudioOverviewResult, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCGetAudioOverview,
		Args: []interface{}{
			projectID,
			1,
		},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("get audio overview: %w", err)
	}

	var data []interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}

	result := &AudioOverviewResult{
		ProjectID: projectID,
	}

	// Handle empty or nil response
	if len(data) == 0 {
		return result, nil
	}

	// Parse the wrb.fr response format
	// Format: [null,null,[3,"<base64-audio>","<id>","<title>",null,true],null,[false]]
	if len(data) > 2 {
		audioData, ok := data[2].([]interface{})
		if !ok || len(audioData) < 4 {
			return nil, fmt.Errorf("invalid audio data format")
		}

		// Extract audio data (index 1)
		if audioBase64, ok := audioData[1].(string); ok {
			result.AudioData = audioBase64
		}

		// Extract ID (index 2)
		if id, ok := audioData[2].(string); ok {
			result.AudioID = id
		}

		// Extract title (index 3)
		if title, ok := audioData[3].(string); ok {
			result.Title = title
		}

		// Extract ready status (index 5)
		if len(audioData) > 5 {
			if ready, ok := audioData[5].(bool); ok {
				result.IsReady = ready
			}
		}
	}

	return result, nil
}

// AudioOverviewResult represents an audio overview response
type AudioOverviewResult struct {
	ProjectID string
	AudioID   string
	Title     string
	AudioData string // Base64 encoded audio data
	IsReady   bool
}

// GetAudioBytes returns the decoded audio data
func (r *AudioOverviewResult) GetAudioBytes() ([]byte, error) {
	if r.AudioData == "" {
		return nil, fmt.Errorf("no audio data available")
	}
	return base64.StdEncoding.DecodeString(r.AudioData)
}

func (c *Client) DeleteAudioOverview(projectID string) error {
	_, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCDeleteAudioOverview,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	return err
}

// Generation operations

func (c *Client) GenerateDocumentGuides(projectID string) (*pb.GenerateDocumentGuidesResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGenerateDocumentGuides,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("generate document guides: %w", err)
	}

	var guides pb.GenerateDocumentGuidesResponse
	if err := beprotojson.Unmarshal(resp, &guides); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &guides, nil
}

func (c *Client) GenerateNotebookGuide(projectID string) (*pb.GenerateNotebookGuideResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGenerateNotebookGuide,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("generate notebook guide: %w", err)
	}

	var guide pb.GenerateNotebookGuideResponse
	if err := beprotojson.Unmarshal(resp, &guide); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &guide, nil
}

func (c *Client) GenerateOutline(projectID string) (*pb.GenerateOutlineResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGenerateOutline,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("generate outline: %w", err)
	}

	var outline pb.GenerateOutlineResponse
	if err := beprotojson.Unmarshal(resp, &outline); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &outline, nil
}

func (c *Client) GenerateSection(projectID string) (*pb.GenerateSectionResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCGenerateSection,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("generate section: %w", err)
	}

	var section pb.GenerateSectionResponse
	if err := beprotojson.Unmarshal(resp, &section); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &section, nil
}

func (c *Client) StartDraft(projectID string) (*pb.StartDraftResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCStartDraft,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("start draft: %w", err)
	}

	var draft pb.StartDraftResponse
	if err := beprotojson.Unmarshal(resp, &draft); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &draft, nil
}

func (c *Client) StartSection(projectID string) (*pb.StartSectionResponse, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID:         rpc.RPCStartSection,
		Args:       []interface{}{projectID},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("start section: %w", err)
	}

	var section pb.StartSectionResponse
	if err := beprotojson.Unmarshal(resp, &section); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &section, nil
}

// Sharing operations

// ShareOption represents audio sharing visibility options
type ShareOption int

const (
	SharePrivate ShareOption = 0
	SharePublic  ShareOption = 1
)

// ShareAudioResult represents the response from sharing audio
type ShareAudioResult struct {
	ShareURL string
	ShareID  string
	IsPublic bool
}

// ShareAudio shares an audio overview with optional public access
func (c *Client) ShareAudio(projectID string, shareOption ShareOption) (*ShareAudioResult, error) {
	resp, err := c.rpc.Do(rpc.Call{
		ID: rpc.RPCShareAudio,
		Args: []interface{}{
			[]int{int(shareOption)},
			projectID,
		},
		NotebookID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("share audio: %w", err)
	}

	// Parse the raw response
	var data []interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	result := &ShareAudioResult{
		IsPublic: shareOption == SharePublic,
	}

	// Extract share URL and ID from response
	if len(data) > 0 {
		if shareData, ok := data[0].([]interface{}); ok && len(shareData) > 0 {
			if shareURL, ok := shareData[0].(string); ok {
				result.ShareURL = shareURL
			}
			if len(shareData) > 1 {
				if shareID, ok := shareData[1].(string); ok {
					result.ShareID = shareID
				}
			}
		}
	}

	return result, nil
}

// Helper functions to identify and extract YouTube video IDs
func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func extractYouTubeVideoID(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	if u.Host == "youtu.be" {
		return strings.TrimPrefix(u.Path, "/"), nil
	}

	if strings.Contains(u.Host, "youtube.com") && u.Path == "/watch" {
		return u.Query().Get("v"), nil
	}

	return "", fmt.Errorf("unsupported YouTube URL format")
}
