((root) => {
  const layout = typeof module !== 'undefined' && module.exports ? require('./layout.js') : root.FamslideLayout;

  function groupPhotos(ordered, width, height, random) {
    const groups = [];
    let bag = [];
    let offset = 0;
    while (offset < ordered.length) {
      const remaining = ordered.length - offset;
      if (!bag.some(size => size <= remaining)) bag = [1, 2, 3, 4];
      const options = bag.filter(size => size <= remaining && (ordered.length <= 2 || size < ordered.length));
      const choices = options.map(size => {
        const photos = ordered.slice(offset, offset + size);
        const quality = layout.evaluateLayout(photos, width, height).quality;
        return {size, weight: Math.pow(quality, 4)};
      });
      let draw = random() * choices.reduce((total, choice) => total + choice.weight, 0);
      let size = choices.at(-1).size;
      for (const choice of choices) {
        draw -= choice.weight;
        if (draw < 0) { size = choice.size; break; }
      }
      groups.push(ordered.slice(offset, offset + size));
      bag = bag.filter(value => value !== size);
      offset += size;
    }
    return groups;
  }

  function prefersGroup(photo, width, height) {
    if (!(photo.width > 0 && photo.height > 0)) return false;
    return (width < height && photo.width > photo.height) ||
      (width > height && photo.height > photo.width);
  }

  function avoidMismatchedSingles(groups, width, height) {
    if (groups.length === 1) return groups;
    for (let i = 0; i < groups.length; i++) {
      if (groups[i].length !== 1 || !prefersGroup(groups[i][0], width, height)) continue;
      // Favor a companion of the same shape, giving the layout scorer a useful
      // stack or row. Orientation takes priority over the group-size variety bag.
      const candidates = groups.map((group, index) => ({group, index}))
        .filter(candidate => candidate.index !== i)
        .sort((a, b) => Number(b.group.some(photo => prefersGroup(photo, width, height))) -
          Number(a.group.some(photo => prefersGroup(photo, width, height))));
      const target = candidates.find(candidate => candidate.group.length < 4);
      if (target) {
        target.group.push(groups[i][0]);
        groups.splice(i--, 1);
      } else {
        // All other groups are full: turn one four and the single into three + two.
        const donor = candidates[0].group;
        const companion = donor.findIndex(photo => prefersGroup(photo, width, height));
        groups[i].push(donor.splice(Math.max(0, companion), 1)[0]);
      }
    }
    return groups;
  }

  function shuffle(items, random) {
    const result = [...items];
    for (let i = result.length - 1; i > 0; i--) {
      const j = Math.floor(random() * (i + 1));
      [result[i], result[j]] = [result[j], result[i]];
    }
    return result;
  }

  function companions(photos) {
    const pairs = [];
    for (let i = 0; i < photos.length; i++) {
      for (let j = i + 1; j < photos.length; j++) pairs.push(groupKey([photos[i], photos[j]]));
    }
    return pairs;
  }

  function groupKey(photos) {
    return photos.map(photo => photo.id).sort().join('\0');
  }

  function buildPlaylist(manifest, previous = [], random = Math.random, viewport = {}) {
    const unique = new Map();
    for (const scene of manifest.scenes || []) {
      for (const photo of scene.photos || []) unique.set(photo.id, photo);
    }
    const photos = [...unique.values()];
    if (!photos.length) return shuffle(manifest.scenes || [], random);

    const priorGroups = new Set(previous.map(scene => groupKey(scene.photos || [])));
    const priorPairs = new Set(previous.flatMap(scene => companions(scene.photos || [])));
    const priorLast = new Set((previous.at(-1)?.photos || []).map(photo => photo.id));
    const width = viewport.width > 0 ? viewport.width : 1920;
    const height = viewport.height > 0 ? viewport.height : 1080;

    let best = null;
    let bestScore = Infinity;
    for (let attempt = 0; attempt < 48; attempt++) {
      // Shift retries so even a repeating random source explores other groupings.
      const attemptRandom = () => (random() + attempt * .618033988749895) % 1;
      const ordered = shuffle(photos, attemptRandom);
      const groups = photos.length === 1 ? [ordered]
        : photos.length === 2 ? (previous.some(scene => scene.photos?.length === 2) ? ordered.map(photo => [photo]) : [ordered])
        : groupPhotos(ordered, width, height, attemptRandom);
      avoidMismatchedSingles(groups, width, height);
      const repeated = groups.filter(group => priorGroups.has(groupKey(group))).length;
      const boundary = groups[0].filter(photo => priorLast.has(photo.id)).length;
      const sharedPairs = groups.flatMap(companions).filter(pair => priorPairs.has(pair)).length;
      const score = repeated * 10 + boundary + sharedPairs / (photos.length * 10);
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
              const mismatchedSingle = [best[i], best[j]].some(group =>
                group.length === 1 && prefersGroup(group[0], width, height));
              if (after < before && !mismatchedSingle) { improved = true; break; }
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
