package gateway

import (
	"net/http"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
	update "github.com/HP-jh/TarsEnterprise/pkg/tarsupdate"
)

func (g *Gateway) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	updater := update.NewUpdater("te", config.Version, "HP-jh", "TarsEnterprise")
	cr := updater.Check()

	resp := map[string]interface{}{
		"has_update": cr.HasUpdate,
		"current":    cr.Current.String(),
		"latest":     cr.Latest.String(),
	}
	if cr.AssetName != "" {
		resp["asset_name"] = cr.AssetName
	}
	if cr.AssetURL != "" {
		resp["asset_url"] = cr.AssetURL
	}
	if cr.Error != nil {
		// GitHub unreachable: fallback to stable_versions.json
		stableVer, err := updater.FallbackToStable()
		if err == nil && stableVer != "" {
			resp["fallback_stable"] = stableVer
			resp["github_error"] = cr.Error.Error()
		} else {
			resp["error"] = cr.Error.Error()
		}
	}
	writeJSON(w, resp)
}

func (g *Gateway) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Admin authorization via X-Admin-Token header
	if g.cfg.AdminToken == "" {
		http.Error(w, "admin token not configured", http.StatusForbidden)
		return
	}
	if r.Header.Get("X-Admin-Token") != g.cfg.AdminToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	updater := update.NewUpdater("te", config.Version, "HP-jh", "TarsEnterprise")
	cr := updater.Check()
	if cr.Error != nil {
		writeJSON(w, map[string]interface{}{
			"ok":    false,
			"error": cr.Error.Error(),
			"note":  "use fallback or retry later",
		})
		return
	}
	if !cr.HasUpdate {
		writeJSON(w, map[string]interface{}{
			"ok":      true,
			"message": "already at latest version",
			"version": cr.Current.String(),
		})
		return
	}

	if err := updater.Apply(cr); err != nil {
		writeJSON(w, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"message": "update applied, restart required",
		"from":    cr.Current.String(),
		"to":      cr.Latest.String(),
	})
}
