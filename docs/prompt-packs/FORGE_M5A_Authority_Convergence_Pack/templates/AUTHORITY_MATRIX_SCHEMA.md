# Authority Matrix Schema Template

```text
AuthorityMatrixRow
  id: string
  surface: string
  method: string
  route: string
  action: string
  authorityOwner: enum
  capabilityId: string
  gatewayCapabilityStatus: enum
  mutating: bool
  destructive: bool
  requiresApproval: bool
  approvalMechanism: enum
  auditCategory: string
  auditAction: string
  liveAuthority: bool
  forgeKAuthority: bool
  hostMutation: bool
  modelruntimeMutation: bool
  semanticMemoryWrite: bool
  status: enum
  notes: string
```

## Example row

```text
id: model.delete_file
surface: modelruntime
method: POST
route: /forge/models/{id}/delete-file
authorityOwner: modelruntime
capabilityId: model.delete_file
gatewayCapabilityStatus: approval_only
mutating: true
destructive: true
requiresApproval: true
approvalMechanism: modelruntime_management
auditCategory: model_runtime
auditAction: model.delete_file
liveAuthority: true
forgeKAuthority: false
hostMutation: false
modelruntimeMutation: true
semanticMemoryWrite: false
status: real
notes: Managed model-home delete only; approval required.
```
