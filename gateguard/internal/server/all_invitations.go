package server

import (
	"context"
	"fmt"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/models"
	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) AllInvitations(ctx context.Context, req *proto.AllInvitationsRequest) (*proto.AllInvitationsResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_inviter":      req.Inviter,
		"request_invitee":      req.Invitee,
		"request_organization": uuid.FromBytesOrNil(req.Organization).String(),
	})
	h.log.DebugCtx(ctx, "AllInvitations handler called")

	invitations, hasMore, err := h.srv.AllInvitations(ctx, &service.AllInvitationsFilter{
		Inviter:      req.Inviter,
		Invitee:      req.Invitee,
		Organization: parseUUID(req.Organization),
		Statuses:     parseStatuses(req.Statuses),
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to get invitations")
		return nil, fmt.Errorf("all invitations: %w", err)
	}

	return &proto.AllInvitationsResponse{
		Invitations: models.InvitationsToProto(invitations),
		HasMore:     hasMore,
	}, nil
}

func parseUUID(id []byte) *uuid.UUID {
	uid := uuid.FromBytesOrNil(id)

	if uid.IsNil() {
		return nil
	}

	return pointer.Ref(uid)
}

func parseStatuses(statuses []proto.InvitationStatus) []models.InvitationStatus {
	result := make([]models.InvitationStatus, len(statuses))

	for i, status := range statuses {
		result[i] = models.InvitationStatus(status)
	}

	return result
}
