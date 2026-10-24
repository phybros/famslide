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


test('mixed albums include singles, pairs, triples and fours in both screen orientations', () => {
  for (const viewport of [{width: 1080, height: 1920}, {width: 1920, height: 1080}]) {
    const source = manifest(40);
    source.scenes.forEach((scene, i) => {
      if (i % 2) Object.assign(scene.photos[0], {width: 900, height: 1600});
    });
    const nextRandom = random(123);
    let previous = [];
    for (let pass = 0; pass < 10; pass++) {
      const playlist = buildPlaylist(source, previous, nextRandom, viewport);
      assert.deepEqual([...new Set(playlist.map(scene => scene.photos.length))].sort(), [1, 2, 3, 4]);
      assert.equal(new Set(playlist.flatMap(scene => scene.photos.map(photo => photo.id))).size, 40);
      for (const scene of playlist.filter(scene => scene.photos.length === 1)) {
        const photo = scene.photos[0];
        assert.equal(photo.width > photo.height, viewport.width > viewport.height);
      }
      previous = playlist;
    }
  }
});

test('viewport shape changes grouping with the same photos and random draws', () => {
  const portrait = buildPlaylist(manifest(40), [], random(123), {width: 1080, height: 1920});
  const landscape = buildPlaylist(manifest(40), [], random(123), {width: 1920, height: 1080});
  assert.notDeepEqual(portrait.map(scene => scene.photos.length), landscape.map(scene => scene.photos.length));
});

test('reshuffling avoids repeating companions inside differently sized groups', () => {
  const nextRandom = random(123);
  const source = manifest(40);
  const first = buildPlaylist(source, [], nextRandom);
  const second = buildPlaylist(source, first, nextRandom);
  function pairs(playlist) {
    return new Set(playlist.flatMap(scene => scene.photos.flatMap((photo, i) =>
      scene.photos.slice(i + 1).map(other => [photo.id, other.id].sort().join(',')))));
  }
  const previousPairs = pairs(first);
  for (const pair of pairs(second)) assert.equal(previousPairs.has(pair), false, `repeated companions ${pair}`);
});


test('opposite-orientation photos are grouped on both screen orientations, including tiny albums', () => {
  const {chooseLayout} = require('./layout.js');
  for (const viewport of [{width: 1080, height: 1920}, {width: 1920, height: 1080}]) {
    for (let count = 2; count <= 41; count++) {
      const source = manifest(count);
      if (viewport.width > viewport.height) {
        source.scenes.forEach(scene => Object.assign(scene.photos[0], {width: 900, height: 1600}));
      }
      const nextRandom = random(count);
      let previous = [];
      for (let pass = 0; pass < 5; pass++) {
        const playlist = buildPlaylist(source, previous, nextRandom, viewport);
        const ids = playlist.flatMap(scene => scene.photos.map(photo => photo.id));
        assert.equal(ids.length, count);
        assert.equal(new Set(ids).size, count);
        for (const scene of playlist) {
          assert.ok(scene.photos.length >= 2 && scene.photos.length <= 4, `count ${count}: mismatched single`);
          const name = chooseLayout(scene.photos, viewport.width, viewport.height);
          assert.ok(name.endsWith(viewport.width < viewport.height ? '-rows' : '-cols'), name);
        }
        previous = playlist;
      }
    }
  }
});

test('a lone opposite-orientation photo remains playable', () => {
  const playlist = buildPlaylist(manifest(1), [], random(1), {width: 1080, height: 1920});
  assert.equal(playlist.length, 1);
  assert.equal(playlist[0].photos.length, 1);
});
