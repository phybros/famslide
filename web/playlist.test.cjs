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

test('every photo appears once and only opposite-orientation photos stack', () => {
  for (const viewport of [{width: 1920, height: 1080}, {width: 1080, height: 1920}]) {
    for (let count = 1; count <= 41; count++) {
      const source = manifest(count);
      source.scenes.forEach((scene, i) => {
        if (i % 2) Object.assign(scene.photos[0], {width: 900, height: 1600});
      });
      const playlist = buildPlaylist(source, [], random(count), viewport);
      const ids = playlist.flatMap(scene => scene.photos.map(photo => photo.id));
      assert.equal(ids.length, count);
      assert.equal(new Set(ids).size, count);
      for (const scene of playlist) {
        assert.ok(scene.photos.length >= 1 && scene.photos.length <= 3);
        if (scene.photos.length > 1) {
          for (const photo of scene.photos) {
            assert.equal(photo.width > photo.height, viewport.width < viewport.height);
          }
        }
      }
    }
  }
});

test('opposite-orientation albums form triples with smaller remainders', () => {
  for (let count = 1; count <= 10; count++) {
    const playlist = buildPlaylist(manifest(count), [], random(1), {width: 1080, height: 1920});
    assert.equal(playlist.length, Math.ceil(count / 3));
    assert.equal(playlist.filter(scene => scene.photos.length === 3).length, Math.floor(count / 3));
  }
});

test('matching orientation, square and unknown dimensions stay single', () => {
  const source = manifest(4);
  Object.assign(source.scenes[0].photos[0], {width: 100, height: 100});
  Object.assign(source.scenes[1].photos[0], {width: 0, height: 0});
  assert.ok(buildPlaylist(source).every(scene => scene.photos.length === 1));
});

test('rotation regroups the same album', () => {
  const source = manifest(6);
  assert.equal(buildPlaylist(source).length, 6);
  assert.equal(buildPlaylist(source, [], random(1), {width: 1080, height: 1920}).length, 2);
});

test('deduplicates photos from cached group manifests', () => {
  const source = manifest(3);
  source.scenes.push({photos: source.scenes.flatMap(scene => scene.photos)});
  assert.equal(buildPlaylist(source).flatMap(scene => scene.photos).length, 3);
});

test('avoids an immediate repeat between passes when possible', () => {
  const source = manifest(6);
  const first = buildPlaylist(source, [], random(1));
  const second = buildPlaylist(source, first, random(2));
  assert.notEqual(first.at(-1).photos[0].id, second[0].photos[0].id);
});

test('an old scene manifest remains playable', () => {
  const source = {scenes: [{id: 'a', image: '/media/a'}, {id: 'b', image: '/media/b'}]};
  assert.deepEqual(buildPlaylist(source, [], random(1)).map(scene => scene.id).sort(), ['a', 'b']);
});
