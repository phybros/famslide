((root) => {
  function shuffle(items, random) {
    const result = [...items];
    for (let i = result.length - 1; i > 0; i--) {
      const j = Math.floor(random() * (i + 1));
      [result[i], result[j]] = [result[j], result[i]];
    }
    return result;
  }

  function groupKey(photos) {
    return photos.map(photo => photo.id).sort().join('\0');
  }

  function buildPlaylist(manifest, previous = [], random = Math.random) {
    const unique = new Map();
    for (const scene of manifest.scenes || []) {
      for (const photo of scene.photos || []) unique.set(photo.id, photo);
    }
    const photos = [...unique.values()];
    if (!photos.length) return shuffle(manifest.scenes || [], random);

    const priorGroups = new Set(previous.map(scene => groupKey(scene.photos || [])));
    const priorLast = new Set((previous.at(-1)?.photos || []).map(photo => photo.id));
    let sizes;
    if (photos.length === 1) {
      sizes = [1];
    } else if (photos.length === 2) {
      sizes = previous.some(scene => scene.photos?.length === 2) ? [1, 1] : [2];
    } else {
      sizes = [];
      let remaining = photos.length;
      if (remaining >= 5 && random() < .35) { sizes.push(3); remaining -= 3; }
      while (remaining >= 2) { sizes.push(2); remaining -= 2; }
      if (remaining) sizes.push(1);
    }

    let best = null;
    let bestScore = Infinity;
    for (let attempt = 0; attempt < 48; attempt++) {
      const ordered = shuffle(photos, random);
      const groups = [];
      let offset = 0;
      for (const size of shuffle(sizes, random)) {
        groups.push(ordered.slice(offset, offset + size));
        offset += size;
      }
      const repeated = groups.filter(group => priorGroups.has(groupKey(group))).length;
      const boundary = groups[0].filter(photo => priorLast.has(photo.id)).length;
      const score = repeated * 10 + boundary;
      if (score < bestScore) { best = groups; bestScore = score; }
      if (score === 0) break;
    }
    // A deterministic fallback keeps pairings fresh even if random draws are unlucky.
    for (let pass = 0; pass < best.length; pass++) {
      let improved = false;
      for (let i = 0; i < best.length && !improved; i++) {
        if (!priorGroups.has(groupKey(best[i]))) continue;
        for (let j = 0; j < best.length && !improved; j++) {
          if (i === j) continue;
          const before = Number(priorGroups.has(groupKey(best[i]))) + Number(priorGroups.has(groupKey(best[j])));
          for (let a = 0; a < best[i].length && !improved; a++) {
            for (let b = 0; b < best[j].length; b++) {
              [best[i][a], best[j][b]] = [best[j][b], best[i][a]];
              const after = Number(priorGroups.has(groupKey(best[i]))) + Number(priorGroups.has(groupKey(best[j])));
              if (after < before) { improved = true; break; }
              [best[i][a], best[j][b]] = [best[j][b], best[i][a]];
            }
          }
        }
      }
      if (!improved) break;
    }
    return best.map(group => ({
      id: groupKey(group),
      photos: group,
    }));
  }

  const api = {buildPlaylist};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FamslidePlaylist = api;
})(globalThis);
