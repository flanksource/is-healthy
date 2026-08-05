local function failureMessage()
  if obj.status.error ~= nil and obj.status.error ~= "" then
    return obj.status.error
  end
  if obj.status.commandError ~= nil and obj.status.commandError ~= "" then
    return obj.status.commandError
  end
  return "CloudNativePG backup failed"
end

if obj.status == nil or obj.status.phase == nil or obj.status.phase == "" then
  return {
    status = "Progressing",
    message = "Waiting for CloudNativePG backup status",
  }
end

local phase = obj.status.phase

if phase == "failed" or phase == "walArchivingFailing" or phase == "invalid backup definition" then
  return {
    status = "Degraded",
    message = failureMessage(),
  }
end

if obj.status.error ~= nil and obj.status.error ~= "" then
  return {
    status = "Degraded",
    message = obj.status.error,
  }
end

if phase == "completed" then
  return {
    status = "Healthy",
    message = "Backup completed",
  }
end

if phase == "pending" or phase == "started" or phase == "running" or phase == "finalizing" then
  return {
    status = "Progressing",
    message = "Backup " .. phase,
  }
end

return {
  status = "Unknown",
  message = "Unknown CloudNativePG backup phase: " .. phase,
}
