function requestServerMachineStats() {
  const machineSelect = document.getElementById("craft-machine");
  if (!(machineSelect instanceof HTMLSelectElement) || machineSelect.value.trim() === "") {
    return;
  }

  const machineQualitySelect = document.getElementById("machine-quality");
  if (!machineQualitySelect) {
    return;
  }

  machineQualitySelect.dispatchEvent(new Event("change", { bubbles: true }));
}

function syncModuleQualityOptionsToMaxUnlocked() {
  const qualitySelects = Array.from(document.querySelectorAll('select[name^="machine_module_slot_"][name$="_quality"]'));
  const allowedQualityOptions = getAllowedModuleQualityOptions();
  const highestAllowedQuality = allowedQualityOptions.length > 0
    ? allowedQualityOptions[allowedQualityOptions.length - 1].value
    : "normal";

  if (qualitySelects.length === 0) {
    requestServerMachineStats();
    return;
  }

  for (const qualitySelect of qualitySelects) {
    const currentValue = String(qualitySelect.value ?? "").toLowerCase();
    qualitySelect.innerHTML = "";

    for (const qualityOption of allowedQualityOptions) {
      const option = document.createElement("option");
      option.value = qualityOption.value;
      option.textContent = qualityOption.label;
      qualitySelect.appendChild(option);
    }

    const nextValue = allowedQualityOptions.some((option) => option.value === currentValue)
      ? currentValue
      : highestAllowedQuality;
    qualitySelect.value = nextValue;

    const slotIndex = qualitySelect.getAttribute("data-slot-index");
    const moduleSelect = slotIndex ? document.getElementById(`machine-module-slot-${slotIndex}`) : null;
    qualitySelect.disabled = !moduleSelect || moduleSelect.value === "";
  }

  requestServerMachineStats();
}

function resetMachineCountFieldsAboveMaxUnlocked() {
  const maxUnlockedQuality = getMaxUnlockedQuality();
  const maxRank = qualityRank.get(maxUnlockedQuality) ?? qualityRank.get("legendary");
  let firstChangedInput = null;

  for (const tier of qualityOrder) {
    const tierRank = qualityRank.get(tier);
    if (tierRank === undefined || tierRank <= maxRank) {
      continue;
    }

    const input = document.getElementById(`${tier}-machines`);
    if (!input) {
      continue;
    }

    if (input.value !== "0") {
      input.value = "0";
      if (!firstChangedInput) {
        firstChangedInput = input;
      }
    }
  }

  if (firstChangedInput) {
    firstChangedInput.dispatchEvent(new Event("change", { bubbles: true }));
  }
}

function syncMachineQualityFromMaxUnlocked() {
  const machineQualitySelect = document.getElementById("machine-quality");
  if (!machineQualitySelect) {
    return;
  }

  const maxUnlockedQuality = getMaxUnlockedQuality();
  const maxRank = qualityRank.get(maxUnlockedQuality) ?? qualityRank.get("legendary");
  const currentValue = String(machineQualitySelect.value ?? "").trim().toLowerCase();

  machineQualitySelect.innerHTML = "";
  let highestAllowedQuality = "normal";
  for (const quality of qualityOrder) {
    const rank = qualityRank.get(quality);
    if (rank === undefined || rank > maxRank) {
      continue;
    }

    const option = document.createElement("option");
    option.value = quality;
    option.textContent = formatQualityLabel(quality);
    machineQualitySelect.appendChild(option);
    highestAllowedQuality = quality;
  }

  const hasCurrent = Array.from(machineQualitySelect.options).some((option) => option.value === currentValue);
  const defaultQuality = Array.from(machineQualitySelect.options).some((option) => option.value === "normal")
    ? "normal"
    : highestAllowedQuality;
  machineQualitySelect.value = hasCurrent ? currentValue : defaultQuality;
}

function setupMachineStatRecalculationDelegation() {
  document.addEventListener("change", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLSelectElement)) {
      return;
    }

    const selectName = String(target.name ?? "");
    const moduleCount = document.querySelectorAll('select[name^="machine_module_slot_"]:not([name$="_quality"])').length;

    const moduleQualityMatch = selectName.match(/^machine_module_slot_(\d+)_quality$/);
    if (moduleQualityMatch) {
      const slotIndex = Number(moduleQualityMatch[1]);
      if (!Number.isFinite(slotIndex) || slotIndex <= 0) {
        return;
      }

      const sourceQualityValue = target.value;
      for (let i = slotIndex + 1; i <= moduleCount; i += 1) {
        const downstreamModule = document.querySelector(`select[name="machine_module_slot_${i}"]`);
        const downstreamQuality = document.querySelector(`select[name="machine_module_slot_${i}_quality"]`);
        if (!(downstreamModule instanceof HTMLSelectElement) || !(downstreamQuality instanceof HTMLSelectElement)) {
          continue;
        }
        if (downstreamModule.value === "") {
          continue;
        }
        const hasQualityOption = Array.from(downstreamQuality.options).some((option) => option.value === sourceQualityValue);
        downstreamQuality.value = hasQualityOption ? sourceQualityValue : "normal";
      }
      return;
    }

    const moduleSlotMatch = selectName.match(/^machine_module_slot_(\d+)$/);
    if (!moduleSlotMatch) {
      return;
    }

    const slotIndex = Number(moduleSlotMatch[1]);
    if (!Number.isFinite(slotIndex) || slotIndex <= 0) {
      return;
    }

    const qualitySelect = document.querySelector(`select[name="machine_module_slot_${slotIndex}_quality"]`);
    if (qualitySelect instanceof HTMLSelectElement) {
      qualitySelect.disabled = target.value === "";
      if (qualitySelect.disabled) {
        qualitySelect.value = "normal";
      }
    }

    // Cascade the module choice (and its quality) to all lower slots.
    const sourceModuleValue = target.value;
    const sourceQualityValue = qualitySelect instanceof HTMLSelectElement ? qualitySelect.value : "normal";
    for (let i = slotIndex + 1; i <= moduleCount; i += 1) {
      const downstreamModule = document.querySelector(`select[name="machine_module_slot_${i}"]`);
      if (!(downstreamModule instanceof HTMLSelectElement)) {
        continue;
      }

      downstreamModule.value = sourceModuleValue;

      const downstreamQuality = document.querySelector(`select[name="machine_module_slot_${i}_quality"]`);
      if (!(downstreamQuality instanceof HTMLSelectElement)) {
        continue;
      }

      downstreamQuality.disabled = sourceModuleValue === "";
      if (downstreamQuality.disabled) {
        downstreamQuality.value = "normal";
        continue;
      }

      const hasQualityOption = Array.from(downstreamQuality.options).some((option) => option.value === sourceQualityValue);
      downstreamQuality.value = hasQualityOption ? sourceQualityValue : "normal";
    }
  }, true);
}
