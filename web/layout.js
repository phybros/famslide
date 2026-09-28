((root) => {
  function containedArea(photo, width, height) {
    const imageRatio = (photo.width || 1) / (photo.height || 1);
    const boxRatio = width / height;
    return width * height * Math.min(imageRatio / boxRatio, boxRatio / imageRatio);
  }

  function chooseLayout(photos, width, height) {
    if (photos.length === 1) return 'single';
    const gap = Math.max(4, Math.round(Math.min(width, height) * .006));
    const halfWidth = (width - gap) / 2;
    const halfHeight = (height - gap) / 2;
    const thirdWidth = (width - 2 * gap) / 3;
    const thirdHeight = (height - 2 * gap) / 3;
    let candidates;
    if (photos.length === 2) {
      candidates = [
        ['pair-rows', [[width, halfHeight], [width, halfHeight]]],
        ['pair-cols', [[halfWidth, height], [halfWidth, height]]],
      ];
    } else {
      const leadWidth = (width - gap) * .45;
      const sideWidth = width - gap - leadWidth;
      const leadHeight = (height - gap) * .55;
      const bottomHeight = height - gap - leadHeight;
      candidates = [
        ['triple-rows', Array(3).fill([width, thirdHeight])],
        ['triple-cols', Array(3).fill([thirdWidth, height])],
        ['triple-lead-left', [[leadWidth, height], [sideWidth, halfHeight], [sideWidth, halfHeight]]],
        ['triple-lead-top', [[width, leadHeight], [halfWidth, bottomHeight], [halfWidth, bottomHeight]]],
      ];
    }
    const sameOrientation = photos.every(photo => photo.height >= photo.width) ? 'pair-cols'
      : photos.every(photo => photo.width > photo.height) ? 'pair-rows' : null;
    return candidates.reduce((best, candidate) => {
      const score = candidate[1].reduce((total, box, i) => total + containedArea(photos[i], box[0], box[1]), 0);
      const preferredScore = score * (candidate[0] === sameOrientation ? 1.01 : 1);
      return preferredScore > best.score ? {name: candidate[0], score: preferredScore} : best;
    }, {name: candidates[0][0], score: -1}).name;
  }

  const api = {chooseLayout};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FamslideLayout = api;
})(globalThis);
