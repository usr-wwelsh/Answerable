package endpoint

import (
	"context"
	"errors"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type checkAvailabilityOutput struct {
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
}

type bookIntakeInput struct {
	Name    string `json:"name" jsonschema:"the requester's name"`
	Contact string `json:"contact,omitempty" jsonschema:"optional phone or email to reach the requester"`
	Need    string `json:"need" jsonschema:"what is being requested, e.g. a shelter bed for two people tonight"`
}

type bookIntakeOutput struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	StatusToken string `json:"statusToken"`
}

type checkIntakeStatusInput struct {
	StatusToken string `json:"statusToken" jsonschema:"the statusToken returned by book_intake"`
}

type checkIntakeStatusOutput struct {
	Status string `json:"status"`
}

func (s *Server) newMCPServer(baseURL string) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "answerable", Version: "0.1.0"}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "check_availability",
		Description: "Check this provider's current published facts: availability, eligibility, hours, and intake process. Read-only.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, checkAvailabilityOutput, error) {
		p := s.currentProvider()
		props := p.Properties
		if props == nil {
			props = map[string]string{}
		}
		return nil, checkAvailabilityOutput{Name: p.Name, Properties: props}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "book_intake",
		Description: "Submit an intake request to this provider for a person or people in need. This queues a request for human review — it does not instantly reserve anything. The response includes a statusToken; use check_intake_status with it later to see whether the request was confirmed or denied, instead of re-submitting.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in bookIntakeInput) (*mcp.CallToolResult, bookIntakeOutput, error) {
		req, message, err := s.bookIntake(baseURL, in.Name, in.Contact, in.Need)
		if err != nil {
			var verr *bookingValidationError
			if errors.As(err, &verr) {
				return nil, bookIntakeOutput{}, err
			}
			return nil, bookIntakeOutput{}, errors.New("failed to queue request")
		}
		return nil, bookIntakeOutput{Status: "queued", Message: message, StatusToken: req.StatusToken}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "check_intake_status",
		Description: "Check whether a previously submitted intake request (from book_intake) has been confirmed or denied yet, using the statusToken it returned. Read-only — safe to call repeatedly.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in checkIntakeStatusInput) (*mcp.CallToolResult, checkIntakeStatusOutput, error) {
		req, err := s.store.Status(in.StatusToken)
		if err != nil {
			return nil, checkIntakeStatusOutput{}, err
		}
		return nil, checkIntakeStatusOutput{Status: string(req.Status)}, nil
	})

	return srv
}

func (s *Server) mcpHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.newMCPServer(baseURL(r))
	}, nil)
}
