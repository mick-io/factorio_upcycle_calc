function requestServerMachineStats() {
  const machineSelect = document.getElementById("craft-machine");
  if (!(machineSelect instanceof HTMLSelectElement) || machineSelect.value.trim() === "") {
    return;
  }

  const machineQualitySelect = document.getElementById("machine-quality");
  if (!(machineQualitySelect instanceof HTMLSelectElement)) {
    return;
  }

  machineQualitySelect.dispatchEvent(new Event("change", { bubbles: true }));
}

function requestServerRecyclerStats() {
  const recyclerQualitySelect = document.getElementById("recycler-quality");
  if (!(recyclerQualitySelect instanceof HTMLSelectElement)) {
    return;
  }

  recyclerQualitySelect.dispatchEvent(new Event("change", { bubbles: true }));
}

function syncModuleQualityOptionsToMaxUnlocked() {
  const qualitySelects = Array.from(document.querySelectorAll('select[name$="_quality"][name*="_module_slot_"]'));
  const allowedQualityOptions = getAllowedModuleQualityOptions();
  const highestAllowedQuality = allowedQualityOptions.length > 0
    ? allowedQualityOptions[allowedQualityOptions.length - 1].value
    : "normal";

  for (const qualitySelect of qualitySelects) {
    if (!(qualitySelect instanceof HTMLSelectElement)) {
      continue;
    }

    const nameMatch = String(qualitySelect.name ?? "").match(/^(machine|recycler)_module_slot_(\d+)_quality$/);
    if (!nameMatch) {
      continue;
    }

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

    const prefix = nameMatch[1];
    const slotIndex = nameMatch[2];
    const moduleSelect = document.querySelector(`select[name="${prefix}_module_slot_${slotIndex}"]`);
    qualitySelect.disabled = !(moduleSelect instanceof HTMLSelectElement) || moduleSelect.value === "";
  }

  requestServerMachineStats();
  requestServerRecyclerStats();
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
    if (!(input instanceof HTMLInputElement)) {
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
  if (!(machineQualitySelect instanceof HTMLSelectElement)) {
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

function syncRecyclerQualityFromMaxUnlocked() {
  const recyclerQualitySelect = document.getElementById("recycler-quality");
  if (!(recyclerQualitySelect instanceof HTMLSelectElement)) {
    return;
  }

  const maxUnlockedQuality = getMaxUnlockedQuality();
  const maxRank = qualityRank.get(maxUnlockedQuality) ?? qualityRank.get("legendary");
  const currentValue = String(recyclerQualitySelect.value ?? "").trim().toLowerCase();

  recyclerQualitySelect.innerHTML = "";
  let highestAllowedQuality = "normal";
  for (const quality of qualityOrder) {
    const rank = qualityRank.get(quality);
    if (rank === undefined || rank > maxRank) {
      continue;
    }

    const option = document.createElement("option");
    option.value = quality;
    option.textContent = formatQualityLabel(quality);
    recyclerQualitySelect.appendChild(option);
    highestAllowedQuality = quality;
  }

  const hasCurrent = Array.from(recyclerQualitySelect.options).some((option) => option.value === currentValue);
  const defaultQuality = Array.from(recyclerQualitySelect.options).some((option) => option.value === "normal")
    ? "normal"
    : highestAllowedQuality;
  recyclerQualitySelect.value = hasCurrent ? currentValue : defaultQuality;
}

function setupMachineStatRecalculationDelegation() {
  document.addEventListener("change", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLSelectElement)) {
      return;
    }

    const selectName = String(target.name ?? "");

    const moduleQualityMatch = selectName.match(/^(machine|recycler)_module_slot_(\d+)_quality$/);
    if (moduleQualityMatch) {
      const prefix = moduleQualityMatch[1];
      const slotIndex = Number(moduleQualityMatch[2]);
      if (!Number.isFinite(slotIndex) || slotIndex <= 0) {
        return;
      }

      const moduleCount = document.querySelectorAll(`select[name^="${prefix}_module_slot_"]:not([name$="_quality"])`).length;
      const sourceQualityValue = target.value;
      for (let i = slotIndex + 1; i <= moduleCount; i += 1) {
        const downstreamModule = document.querySelector(`select[name="${prefix}_module_slot_${i}"]`);
        const downstreamQuality = document.querySelector(`select[name="${prefix}_module_slot_${i}_quality"]`);
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

    const moduleSlotMatch = selectName.match(/^(machine|recycler)_module_slot_(\d+)$/);
    if (!moduleSlotMatch) {
      return;
    }

    const prefix = moduleSlotMatch[1];
    const slotIndex = Number(moduleSlotMatch[2]);
    if (!Number.isFinite(slotIndex) || slotIndex <= 0) {
      return;
    }

    const moduleCount = document.querySelectorAll(`select[name^="${prefix}_module_slot_"]:not([name$="_quality"])`).length;

    const qualitySelect = document.querySelector(`select[name="${prefix}_module_slot_${slotIndex}_quality"]`);
    if (qualitySelect instanceof HTMLSelectElement) {
      qualitySelect.disabled = target.value === "";
      if (qualitySelect.disabled) {
        qualitySelect.value = "normal";
      }
    }

    const sourceModuleValue = target.value;
    const sourceQualityValue = qualitySelect instanceof HTMLSelectElement ? qualitySelect.value : "normal";
    for (let i = slotIndex + 1; i <= moduleCount; i += 1) {
      const downstreamModule = document.querySelector(`select[name="${prefix}_module_slot_${i}"]`);
      if (!(downstreamModule instanceof HTMLSelectElement)) {
        continue;
      }

      downstreamModule.value = sourceModuleValue;

      const downstreamQuality = document.querySelector(`select[name="${prefix}_module_slot_${i}_quality"]`);
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
