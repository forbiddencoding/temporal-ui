package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/temporalio/ui-server/v2/server/api"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

// MockWorkflowService is a mock implementation of IWorkflowService
type MockWorkflowService struct {
	mock.Mock
}

func (m *MockWorkflowService) GetWorkflowExecutionHistory(
	ctx context.Context,
	req *workflowservice.GetWorkflowExecutionHistoryRequest,
	opts ...grpc.CallOption,
) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*workflowservice.GetWorkflowExecutionHistoryResponse), args.Error(1)
}

func TestWorkflowRawHistoryHandler_HappyPath(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetParamNames("namespace", "workflow", "runid")
	c.SetParamValues("test-namespace", "test-workflow", "test-runid")

	mockService := new(MockWorkflowService)
	mockService.On("GetWorkflowExecutionHistory", mock.Anything, mock.Anything).
		Return(&workflowservice.GetWorkflowExecutionHistoryResponse{
			History: &history.History{
				Events: []*history.HistoryEvent{
					{EventId: 1, EventType: 1},
				},
			},
		}, nil)

	handler := api.WorkflowRawHistoryHandler(mockService)

	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[{"Attributes": null, "event_id":1, "event_type":1}]`, rec.Body.String())
	mockService.AssertExpectations(t)
}

func TestWorkflowRawHistoryHandler_URLDecoding(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// URL encoded workflow ID (this is "test/workflow+id")
	c.SetParamNames("namespace", "workflow", "runid")
	c.SetParamValues("test-namespace", "test%2Fworkflow%2Bid", "test-runid")

	mockService := new(MockWorkflowService)
	// Check that the decoded workflow ID is passed to the service
	mockService.On("GetWorkflowExecutionHistory", mock.Anything, mock.MatchedBy(func(req *workflowservice.GetWorkflowExecutionHistoryRequest) bool {
		return req.Execution.WorkflowId == "test/workflow+id"
	})).Return(&workflowservice.GetWorkflowExecutionHistoryResponse{
		History: &history.History{
			Events: []*history.HistoryEvent{
				{EventId: 1, EventType: 1},
			},
		},
	}, nil)

	handler := api.WorkflowRawHistoryHandler(mockService)

	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	mockService.AssertExpectations(t)
}

func TestWorkflowRawHistoryHandler_InvalidURLEncoding(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Invalid URL encoding with incomplete percent encoding
	c.SetParamNames("namespace", "workflow", "runid")
	c.SetParamValues("test-namespace", "invalid%2", "test-runid")

	mockService := new(MockWorkflowService)
	// The service should not be called because decoding fails
	handler := api.WorkflowRawHistoryHandler(mockService)

	err := handler(c)

	// Error should be returned
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	assert.Contains(t, httpErr.Message, "Invalid workflow ID")
	// No calls to the service should be made
	mockService.AssertNotCalled(t, "GetWorkflowExecutionHistory")
}

func TestRawHistory_StreamEvents_ErrorHandling(t *testing.T) {
	mockService := new(MockWorkflowService)
	mockService.On("GetWorkflowExecutionHistory", mock.Anything, mock.Anything).
		Return(&workflowservice.GetWorkflowExecutionHistoryResponse{
			History: &history.History{
				Events: []*history.HistoryEvent{
					{EventId: 1, EventType: 1},
				},
			},
		}, fmt.Errorf("fetch error"))

	rec := httptest.NewRecorder()
	rh := api.NewRawHistory().
		SetIDs("test-namespace", "test-workflow", "test-runid").
		SetContext(context.Background()).
		SetWriter(rec).
		SetWorkflowService(mockService)

	err := rh.StreamEvents()

	assert.NoError(t, err)
	assert.JSONEq(t, `[{"error": "fetch error"}]`, rec.Body.String())
	mockService.AssertExpectations(t)
}

func TestRawHistory_StreamEvents_HappyPath(t *testing.T) {
	mockService := new(MockWorkflowService)
	mockService.On("GetWorkflowExecutionHistory", mock.Anything, mock.Anything).
		Return(&workflowservice.GetWorkflowExecutionHistoryResponse{
			History: &history.History{
				Events: []*history.HistoryEvent{
					{EventId: 1, EventType: 1},
					{EventId: 2, EventType: 5},
				},
			},
			NextPageToken: nil,
		}, nil)

	rec := httptest.NewRecorder()
	rh := api.NewRawHistory().
		SetIDs("test-namespace", "test-workflow", "test-runid").
		SetContext(context.Background()).
		SetWriter(rec).
		SetWorkflowService(mockService)

	err := rh.StreamEvents()

	assert.NoError(t, err)
	assert.JSONEq(
		t,
		`[{"Attributes": null, "event_id":1,"event_type":1},{"Attributes": null, "event_id":2,"event_type":5}]`,
		rec.Body.String(),
	)
	mockService.AssertExpectations(t)
}
