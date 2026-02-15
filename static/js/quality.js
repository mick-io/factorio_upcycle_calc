function formatQualityLabel(value) {
  const normalized = String(value ?? "").trim().toLowerCase();
  if (!normalized) {
    return "Normal";
  }
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}

const moduleQualityOptions = [
  { value: "normal", label: "Normal" },
  { value: "uncommon", label: "Uncommon" },
  { value: "rare", label: "Rare" },
  { value: "epic", label: "Epic" },
  { value: "legendary", label: "Legendary" },
];

const qualityOrder = ["normal", "uncommon", "rare", "epic", "legendary"];
const qualityRank = new Map(qualityOrder.map((quality, index) => [quality, index]));
const qualityMultiplierByTier = {
  normal: 1.0,
  uncommon: 1.3,
  rare: 1.6,
  epic: 1.9,
  legendary: 2.5,
};

function getMaxUnlockedQuality() {
  const maxUnlockedInput = document.getElementById("max-quality-unlocked");
  const value = String(maxUnlockedInput?.value ?? "legendary").trim().toLowerCase();
  return qualityRank.has(value) ? value : "legendary";
}

function getAllowedModuleQualityOptions() {
  const maxUnlockedQuality = getMaxUnlockedQuality();
  const maxRank = qualityRank.get(maxUnlockedQuality) ?? qualityRank.get("legendary");
  return moduleQualityOptions.filter((option) => (qualityRank.get(option.value) ?? -1) <= maxRank);
}
