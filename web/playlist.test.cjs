const test = require('node:test');
const assert = require('node:assert/strict');
const {buildPlaylist} = require('./playlist.js');

function random(seed) {
  return () => {
    seed = (seed * 1664525 + 1013904223) >>> 0;
    return seed / 0x100000000;
  };
}

function manifest(count) {
  return {
    scenes: Array.from({length: count}, (_, i) => ({
      id: `fixed-${i}`,
      photos: [{id: String(i), url: `/media/${i}`, width: 1600, height: 900}],
    })),
  };
}

function keys(playlist) {
  return new Set(playlist.map(scene => scene.photos.map(photo => photo.id).sort().join(',')));
}

test('each pass uses every photo exactly once and changes groups', () => {
  for (let count = 3; count <= 12; count++) {
    const source = manifest(count);
    const nextRandom = random(count);
    let previous = [];
    for (let pass = 0; pass < 20; pass++) {
      const playlist = buildPlaylist(source, previous, nextRandom);
      const ids = playlist.flatMap(scene => scene.photos.map(photo => photo.id));
      assert.equal(ids.length, count);
      assert.equal(new Set(ids).size, count);
      if (previous.length) {
        for (const key of keys(playlist)) {
          assert.equal(keys(previous).has(key), false, `count ${count}, pass ${pass}: repeated ${key}`);
        }
      }
      previous = playlist;
    }
  }
});

test('two photos alternate between one pair and two singles', () => {
  const source = manifest(2);
  const nextRandom = random(42);
  const pair = buildPlaylist(source, [], nextRandom);
  assert.deepEqual(pair.map(scene => scene.photos.length), [2]);
  const singles = buildPlaylist(source, pair, nextRandom);
  assert.deepEqual(singles.map(scene => scene.photos.length), [1, 1]);
  assert.deepEqual(buildPlaylist(source, singles, nextRandom).map(scene => scene.photos.length), [2]);
});

test('pairings change even when random draws repeat', () => {
  for (let count = 3; count <= 10; count++) {
    const source = manifest(count);
    const first = buildPlaylist(source, [], () => 0);
    const second = buildPlaylist(source, first, () => 0);
    for (const key of keys(second)) {
      assert.equal(keys(first).has(key), false, `count ${count}: repeated ${key}`);
    }
  }
});

test('an old scene manifest remains playable', () => {
  const source = {scenes: [{id: 'a', image: '/media/a'}, {id: 'b', image: '/media/b'}]};
  assert.deepEqual(buildPlaylist(source, [], random(1)).map(scene => scene.id).sort(), ['a', 'b']);
});
