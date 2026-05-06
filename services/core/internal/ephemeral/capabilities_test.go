package ephemeral

import "testing"

func TestRedisRolesAreEphemeralOnly(t *testing.T) {
	for _, role := range AllowedRoles() {
		if err := ValidateRole(role); err != nil {
			t.Fatalf("expected allowed role %q to validate: %v", role, err)
		}
		if CanBeCanonical(role) {
			t.Fatalf("redis role %q must not be canonical", role)
		}
		if !RedisLossRecoverable(role) {
			t.Fatalf("redis role %q must be recoverable after loss", role)
		}
	}
}

func TestRedisForbiddenRolesRejected(t *testing.T) {
	for _, role := range ForbiddenRoles() {
		if err := ValidateRole(role); err == nil {
			t.Fatalf("expected forbidden role %q to fail", role)
		}
		if CanBeCanonical(role) {
			t.Fatalf("forbidden role %q must not be canonical", role)
		}
		if RedisLossRecoverable(role) {
			t.Fatalf("forbidden role %q must not be modeled as recoverable redis use", role)
		}
	}
}

func TestRedisUnknownRoleRejected(t *testing.T) {
	if err := ValidateRole(Role("durable_job_queue")); err == nil {
		t.Fatalf("expected unknown role to fail")
	}
}
