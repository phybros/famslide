((root) => {
  function shuffle(items, random) {
    const result = [...items];
    for (let i = result.length - 1; i > 0; i--) {
      const j = Math.floor(random() * (i + 1));
      [result[i], result[j]] = [result[j], result[i]];
    }
    return result;
  }

  function buildPlaylist(manifest, previous = [], random = Math.random, viewport = {}) {
    const unique = new Map();
    for (const scene of manifest.scenes || []) {
      for (const photo of scene.photos || []) unique.set(photo.id, photo);
    }
    if (!unique.size) return shuffle(manifest.scenes || [], random);

    const landscape = (viewport.width || 1920) >= (viewport.height || 1080);
    const groups = [];
    const stack = [];
    for (const photo of shuffle([...unique.values()], random)) {
      const opposite = photo.width > 0 && photo.height > 0 &&
        (landscape ? photo.height > photo.width : photo.width > photo.height);
      if (opposite) stack.push(photo);
      else groups.push([photo]);
    }
    for (let i = 0; i < stack.length; i += 3) groups.push(stack.slice(i, i + 3));
    const scenes = shuffle(groups, random).map(photos => ({
      id: photos.map(photo => photo.id).sort().join('\0'),
      photos,
    }));
    // Avoid showing a photo again immediately at the boundary between passes.
    const last = new Set((previous.at(-1)?.photos || []).map(photo => photo.id));
    const first = scenes.findIndex(scene => scene.photos.every(photo => !last.has(photo.id)));
    if (first > 0) [scenes[0], scenes[first]] = [scenes[first], scenes[0]];
    return scenes;
  }

  const api = {buildPlaylist};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FamslidePlaylist = api;
})(globalThis);
