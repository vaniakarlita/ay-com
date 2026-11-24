package handler

import (
	"context"
	"fmt"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
)

type VisibilityDecision struct {
	Allowed bool
	Reason  string
}

func (h *ThreadHandler) CheckContentVisibility(ctx context.Context, viewerID, authorID string) (VisibilityDecision, error) {
	if viewerID == authorID {
		return VisibilityDecision{Allowed: true}, nil
	}

	if viewerID != "" && authorID != "" {
		blkRes, err := h.userClient.CheckBlockStatus(ctx, &userpb.CheckBlockRequest{
			UserA: viewerID,
			UserB: authorID,
		})
		if err == nil && blkRes != nil {
			if blkRes.IsBlockedBy {
				return VisibilityDecision{Allowed: false, Reason: "blocked_by"}, nil
			}
			if blkRes.IsBlocking {
				return VisibilityDecision{Allowed: false, Reason: "blocking"}, nil
			}
		}
	}

	authorProf, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: authorID})
	if err != nil {
		return VisibilityDecision{Allowed: false, Reason: "not_found"}, fmt.Errorf("author not found")
	}

	if authorProf.IsPrivate {
		if viewerID == "" {
			return VisibilityDecision{Allowed: false, Reason: "private"}, nil
		}
		stats, sErr := h.userClient.GetFollowStats(ctx, &userpb.GetFollowStatsRequest{
			TargetUserId: authorID,
			ViewerUserId: viewerID,
		})
		if sErr != nil || !stats.IsFollowing {
			return VisibilityDecision{Allowed: false, Reason: "private"}, nil
		}
	}

	return VisibilityDecision{Allowed: true}, nil
}

func (h *ThreadHandler) FilterVisibleAuthorIDs(ctx context.Context, viewerID string, authorIDs []string) (map[string]bool, error) {
	visibleMap := make(map[string]bool)
	if len(authorIDs) == 0 {
		return visibleMap, nil
	}

	uniqueAuthors := make(map[string]bool)
	var filteredIDs []string
	for _, id := range authorIDs {
		if id == "" {
			continue
		}
		if id == viewerID {
			visibleMap[id] = true
			continue
		}
		if !uniqueAuthors[id] {
			uniqueAuthors[id] = true
			filteredIDs = append(filteredIDs, id)
		}
	}

	if len(filteredIDs) == 0 {
		return visibleMap, nil
	}

	blockedMap := make(map[string]bool)
	followingMap := make(map[string]bool)

	if viewerID != "" {
		blkRes, err := h.userClient.GetBlockedUserIDs(ctx, &userpb.GetBlockedUserIDsRequest{UserId: viewerID})
		if err == nil && blkRes != nil {
			for _, bID := range blkRes.BlockedUserIds {
				blockedMap[bID] = true
			}
		}

		flwRes, err := h.userClient.ListFollowingIDs(ctx, &userpb.ListFollowingIDsRequest{UserId: viewerID, Limit: 500})
		if err == nil && flwRes != nil {
			for _, fID := range flwRes.FollowingIds {
				followingMap[fID] = true
			}
		}
	}

	activeRes, err := h.authClient.GetActiveProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: filteredIDs})
	if err != nil || activeRes == nil {
		return visibleMap, err
	}

	for _, prof := range activeRes.Profiles {
		if blockedMap[prof.UserId] {
			continue
		}
		if prof.IsPrivate {
			if viewerID != "" && followingMap[prof.UserId] {
				visibleMap[prof.UserId] = true
			}
		} else {
			visibleMap[prof.UserId] = true
		}
	}

	return visibleMap, nil
}
