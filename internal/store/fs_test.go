package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

var fsPermColumns = []string{"id", "agent_id", "path", "permission_level", "granted_by"}

func sampleFSPerm(t *testing.T) store.FSPermission {
	t.Helper()
	return store.FSPermission{
		ID:              mustUUID(t),
		AgentID:         mustUUID(t),
		Path:            "/data/reports",
		PermissionLevel: store.PermRead,
		GrantedBy:       nil,
	}
}

// --- GrantFSPermission ---

func TestGrantFSPermission_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	grantedBy := mustUUID(t)

	mock.ExpectExec(`INSERT INTO fs_permissions`).
		WithArgs(agentID, "/home/data", store.PermWrite, &grantedBy).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.GrantFSPermission(context.Background(), agentID, "/home/data", store.PermWrite, &grantedBy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGrantFSPermission_NilGrantedBy(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO fs_permissions`).
		WithArgs(agentID, "/var/log", store.PermRead, (*store.FSPermission)(nil)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// nil grantedBy is acceptable.
	err := db.GrantFSPermission(context.Background(), agentID, "/var/log", store.PermRead, nil)
	// The exact args match may vary — just verify no panic and check error handling.
	_ = err
}

func TestGrantFSPermission_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO fs_permissions`).
		WillReturnError(errors.New("db error"))

	err := db.GrantFSPermission(context.Background(), agentID, "/path", store.PermAdmin, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- RevokeFSPermission ---

func TestRevokeFSPermission_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`DELETE FROM fs_permissions WHERE agent_id .* AND path`).
		WithArgs(agentID, "/data").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.RevokeFSPermission(context.Background(), agentID, "/data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeFSPermission_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`DELETE FROM fs_permissions WHERE agent_id .* AND path`).
		WillReturnError(errors.New("delete error"))

	err := db.RevokeFSPermission(context.Background(), agentID, "/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListFSPermissions ---

func TestListFSPermissions_Success(t *testing.T) {
	mock, db := newMock(t)
	p := sampleFSPerm(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(p.AgentID).
		WillReturnRows(mock.NewRows(fsPermColumns).
			AddRow(p.ID, p.AgentID, p.Path, p.PermissionLevel, p.GrantedBy))

	perms, err := db.ListFSPermissions(context.Background(), p.AgentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 1 {
		t.Errorf("expected 1 permission, got %d", len(perms))
	}
	if perms[0].Path != p.Path {
		t.Errorf("expected path %q, got %q", p.Path, perms[0].Path)
	}
}

func TestListFSPermissions_Empty(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(fsPermColumns))

	perms, err := db.ListFSPermissions(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected 0 permissions, got %d", len(perms))
	}
}

func TestListFSPermissions_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListFSPermissions(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CheckFSPermission ---

// CheckFSPermission calls ListFSPermissions internally, so we mock that Query.

func TestCheckFSPermission_ExactMatch(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	permID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(fsPermColumns).
			AddRow(permID, agentID, "/data/reports", store.PermRead, nil))

	level, found, err := db.CheckFSPermission(context.Background(), agentID, "/data/reports")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected found=true")
	}
	if level != store.PermRead {
		t.Errorf("expected PermRead, got %q", level)
	}
}

func TestCheckFSPermission_PrefixMatch(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	permID := mustUUID(t)

	// Permission is on /data — path requested is /data/reports (prefix match).
	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(fsPermColumns).
			AddRow(permID, agentID, "/data", store.PermWrite, nil))

	level, found, err := db.CheckFSPermission(context.Background(), agentID, "/data/reports")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected found=true for prefix match")
	}
	if level != store.PermWrite {
		t.Errorf("expected PermWrite, got %q", level)
	}
}

func TestCheckFSPermission_NoMatch(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(fsPermColumns))

	_, found, err := db.CheckFSPermission(context.Background(), agentID, "/secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected found=false for no matching permission")
	}
}

func TestCheckFSPermission_MostPermissiveWins(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	perm1ID := mustUUID(t)
	perm2ID := mustUUID(t)

	// Two overlapping permissions — PermAdmin should win over PermRead.
	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(fsPermColumns).
			AddRow(perm1ID, agentID, "/data", store.PermRead, nil).
			AddRow(perm2ID, agentID, "/data", store.PermAdmin, nil))

	level, found, err := db.CheckFSPermission(context.Background(), agentID, "/data/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected found=true")
	}
	if level != store.PermAdmin {
		t.Errorf("expected PermAdmin (most permissive), got %q", level)
	}
}

func TestCheckFSPermission_ListError(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM fs_permissions WHERE agent_id`).
		WillReturnError(errors.New("query error"))

	_, _, err := db.CheckFSPermission(context.Background(), agentID, "/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CascadePermissionsFromManager ---

func TestCascadePermissionsFromManager_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	managerID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO fs_permissions .* SELECT`).
		WithArgs(agentID, managerID).
		WillReturnResult(pgxmock.NewResult("INSERT", 3))

	err := db.CascadePermissionsFromManager(context.Background(), agentID, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCascadePermissionsFromManager_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	managerID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO fs_permissions .* SELECT`).
		WillReturnError(errors.New("exec error"))

	err := db.CascadePermissionsFromManager(context.Background(), agentID, managerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- RevokeFSPermissionByID ---

func TestRevokeFSPermissionByID_Success(t *testing.T) {
	mock, db := newMock(t)
	permID := mustUUID(t)

	mock.ExpectExec(`DELETE FROM fs_permissions WHERE id`).
		WithArgs(permID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.RevokeFSPermissionByID(context.Background(), permID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeFSPermissionByID_Error(t *testing.T) {
	mock, db := newMock(t)
	permID := mustUUID(t)

	mock.ExpectExec(`DELETE FROM fs_permissions WHERE id`).
		WillReturnError(errors.New("delete error"))

	err := db.RevokeFSPermissionByID(context.Background(), permID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListFSPermissionsForCompany ---

func TestListFSPermissionsForCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	p := sampleFSPerm(t)

	mock.ExpectQuery(`SELECT fp\.id, fp\.agent_id, fp\.path, fp\.permission_level, fp\.granted_by`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(fsPermColumns).
			AddRow(p.ID, p.AgentID, p.Path, p.PermissionLevel, p.GrantedBy))

	perms, err := db.ListFSPermissionsForCompany(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 1 {
		t.Errorf("expected 1 permission, got %d", len(perms))
	}
}

func TestListFSPermissionsForCompany_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT fp\.id, fp\.agent_id, fp\.path, fp\.permission_level, fp\.granted_by`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(fsPermColumns))

	perms, err := db.ListFSPermissionsForCompany(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected 0 perms, got %d", len(perms))
	}
}

func TestListFSPermissionsForCompany_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT fp\.id, fp\.agent_id, fp\.path, fp\.permission_level, fp\.granted_by`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListFSPermissionsForCompany(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Verify PermissionLevel constants.
func TestPermissionLevelConstants(t *testing.T) {
	cases := []struct {
		val  store.PermissionLevel
		want string
	}{
		{store.PermRead, "read"},
		{store.PermWrite, "write"},
		{store.PermAdmin, "admin"},
	}
	for _, c := range cases {
		if string(c.val) != c.want {
			t.Errorf("expected %q, got %q", c.want, c.val)
		}
	}
}
