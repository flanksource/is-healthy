local function findCondition(conditionType)
  if obj.status == nil or obj.status.conditions == nil then
    return nil
  end

  for _, condition in ipairs(obj.status.conditions) do
    if condition.type == conditionType then
      return condition
    end
  end

  return nil
end

local function conditionIsStale(condition)
  return condition ~= nil
      and condition.observedGeneration ~= nil
      and obj.metadata ~= nil
      and obj.metadata.generation ~= nil
      and condition.observedGeneration < obj.metadata.generation
end

local function conditionMessage(condition, fallback)
  if condition ~= nil then
    if condition.message ~= nil and condition.message ~= "" then
      return condition.message
    end
    if condition.reason ~= nil and condition.reason ~= "" then
      return condition.reason
    end
  end
  return fallback
end

local phaseStatus = {
  ["Cluster in healthy state"] = "Healthy",
  ["Setting up primary"] = "Progressing",
  ["Creating a new replica"] = "Progressing",
  ["Upgrading cluster"] = "Progressing",
  ["Waiting for the instances to become active"] = "Progressing",
  ["Promoting to primary cluster"] = "Progressing",
  ["Switchover in progress"] = "Degraded",
  ["Failing over"] = "Degraded",
  ["Upgrading Postgres major version"] = "Degraded",
  ["Cluster upgrade delayed"] = "Degraded",
  ["Waiting for user action"] = "Degraded",
  ["Primary instance is being restarted in-place"] = "Degraded",
  ["Primary instance is being restarted without a switchover"] = "Degraded",
  ["Online upgrade in progress"] = "Degraded",
  ["Applying configuration"] = "Degraded",
  ["Cluster cannot proceed to reconciliation due to an unknown plugin being required"] = "Degraded",
  ["Cluster cannot proceed to reconciliation due to an error while interacting with plugins"] = "Degraded",
  ["Cluster has incomplete or invalid image catalog"] = "Degraded",
  ["Cluster is unrecoverable and needs manual intervention"] = "Degraded",
  ["Cluster cannot execute instance online upgrade due to missing architecture binary"] = "Degraded",
  ["Unable to create required cluster objects"] = "Degraded",
  ["Invalid cluster definition"] = "Degraded",
}

if obj.metadata ~= nil and obj.metadata.annotations ~= nil
    and obj.metadata.annotations["cnpg.io/reconciliationLoop"] == "disabled" then
  return {
    status = "Suspended",
    message = "Cluster reconciliation is suspended",
  }
end

local hibernation = findCondition("cnpg.io/hibernation")
if hibernation ~= nil and hibernation.status == "True" then
  return {
    status = "Suspended",
    message = conditionMessage(hibernation, "Cluster is hibernated"),
  }
end

if obj.status == nil then
  return {
    status = "Progressing",
    message = "Waiting for CloudNativePG cluster status",
  }
end

local ready = findCondition("Ready")
local hs = {}

if conditionIsStale(ready) then
  hs.status = "Progressing"
  hs.message = "Waiting for CloudNativePG to reconcile the latest generation"
elseif ready ~= nil and ready.status == "True" then
  hs.status = "Healthy"
  hs.message = conditionMessage(ready, obj.status.phaseReason or obj.status.phase)
elseif ready ~= nil and ready.status == "False" then
  hs.status = phaseStatus[obj.status.phase] or "Progressing"
  if hs.status == "Healthy" then
    hs.status = "Degraded"
  end
  hs.message = obj.status.phaseReason
  if hs.message == nil or hs.message == "" then
    hs.message = conditionMessage(ready, obj.status.phase or "Cluster is not ready")
  end
elseif ready ~= nil then
  hs.status = "Progressing"
  hs.message = conditionMessage(ready, obj.status.phaseReason or obj.status.phase)
else
  hs.status = phaseStatus[obj.status.phase] or "Progressing"
  hs.message = obj.status.phaseReason
  if hs.message == nil or hs.message == "" then
    hs.message = obj.status.phase or "Waiting for Cluster readiness condition"
  end
end

local lastBackup = findCondition("LastBackupSucceeded")
if lastBackup ~= nil and lastBackup.status == "False" and lastBackup.reason ~= "BackupStarted"
    and not conditionIsStale(lastBackup) and hs.status ~= "Degraded" and hs.status ~= "Suspended" then
  return {
    ready = (ready ~= nil and ready.status == "True") or hs.status == "Healthy",
    health = "Warning",
    status = lastBackup.reason or "LastBackupFailed",
    message = conditionMessage(lastBackup, "Last backup failed"),
  }
end

return hs
