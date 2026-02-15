function syncTargetQualityOptionsToMaxUnlocked() {
  const targetSelect = document.getElementById("target-quality");
  if (!targetSelect) {
    return;
  }

  const maxUnlockedQuality = getMaxUnlockedQuality();
  const maxRank = qualityRank.get(maxUnlockedQuality) ?? qualityRank.get("legendary");
  let highestAllowedQuality = maxUnlockedQuality;

  for (const option of Array.from(targetSelect.options)) {
    const qualityValue = String(option.value ?? "").trim().toLowerCase();
    const rank = qualityRank.get(qualityValue);
    const isAllowed = rank !== undefined && rank <= maxRank;
    option.disabled = !isAllowed;
    option.hidden = !isAllowed;
    if (isAllowed) {
      highestAllowedQuality = qualityValue;
    }
  }

  const selectedValue = String(targetSelect.value ?? "").trim().toLowerCase();
  const selectedRank = qualityRank.get(selectedValue);
  if (selectedRank === undefined || selectedRank > maxRank) {
    targetSelect.value = highestAllowedQuality;
  }
}

function setupMaxQualityDropdown() {
  const maxUnlockedSelect = document.getElementById("max-quality-unlocked");
  if (!maxUnlockedSelect) {
    return;
  }

  const syncFromMaxUnlocked = () => {
    syncTargetQualityOptionsToMaxUnlocked();
    syncMachineQualityFromMaxUnlocked();
    syncRecyclerQualityFromMaxUnlocked();
    syncModuleQualityOptionsToMaxUnlocked();
    resetMachineCountFieldsAboveMaxUnlocked();
  };

  maxUnlockedSelect.addEventListener("change", syncFromMaxUnlocked);
  syncFromMaxUnlocked();
}
