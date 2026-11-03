((root) => {
  function chooseLayout(photos, width, height) {
    if (photos.length <= 1) return 'single';
    return width >= height ? 'stack-cols' : 'stack-rows';
  }

  // object-position percentages align points in the image and its box. Convert
  // the desired crop offset rather than treating the face center as a percentage.
  function faceCrop(photo, width, height) {
    const b = photo.face_bounds;
    if (!b || !(width > 0 && height > 0 && photo.width > 0 && photo.height > 0)) return null;
    if (![b.x, b.y, b.width, b.height].every(Number.isFinite) ||
        b.x < 0 || b.y < 0 || b.width <= 0 || b.height <= 0 ||
        b.x + b.width > 1.000001 || b.y + b.height > 1.000001) return null;
    const ratio = (width / height) / (photo.width / photo.height);
    const cw = Math.min(1, ratio);
    const ch = Math.min(1, 1 / ratio);
    if (b.width > cw + 1e-9 || b.height > ch + 1e-9) {
      return {fit: 'contain', x: 50, y: 50};
    }
    function position(start, size, visible) {
      if (visible >= 1) return 50;
      const low = Math.max(0, start + size - visible);
      const high = Math.min(start, 1 - visible);
      const offset = Math.max(low, Math.min(high, start + size / 2 - visible / 2));
      return 100 * offset / (1 - visible);
    }
    return {fit: 'cover', x: position(b.x, b.width, cw), y: position(b.y, b.height, ch)};
  }

  const api = {chooseLayout, faceCrop};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FamslideLayout = api;
})(globalThis);
