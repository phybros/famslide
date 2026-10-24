const test = require('node:test');
const assert = require('node:assert/strict');
const {chooseLayout} = require('./layout.js');

test('one photo fills the viewport', () => {
  assert.equal(chooseLayout([{width: 1600, height: 900}], 1080, 1920), 'single');
});

test('pairs rearrange when the viewport rotates', () => {
  const landscape = [{width: 1600, height: 900}, {width: 1600, height: 900}];
  const portrait = [{width: 900, height: 1600}, {width: 900, height: 1600}];
  const mixed = [{width: 900, height: 1600}, {width: 1600, height: 900}];
  assert.equal(chooseLayout(landscape, 1080, 1920), 'pair-rows');
  assert.equal(chooseLayout(landscape, 1920, 1080), 'pair-rows');
  assert.equal(chooseLayout(portrait, 1080, 1920), 'pair-cols');
  assert.equal(chooseLayout(portrait, 1920, 1080), 'pair-cols');
  assert.equal(chooseLayout(mixed, 1080, 1920), 'pair-rows');
  assert.equal(chooseLayout(mixed, 1920, 1080), 'pair-cols');
});

test('mixed groups use the available viewport shape', () => {
  const photos = [
    {width: 900, height: 1600},
    {width: 1600, height: 900},
    {width: 1600, height: 900},
  ];
  assert.equal(chooseLayout(photos, 1080, 1920), 'triple-rows');
  assert.equal(chooseLayout(photos, 1920, 1080), 'triple-lead-left');
});

const {faceCrop} = require('./layout.js');
test('face crops keep an off-center group visible', () => {
  const photo = {width: 2000, height: 1000, face_bounds: {x: .7, y: .2, width: .25, height: .4}};
  const crop = faceCrop(photo, 1000, 1000);
  assert.equal(crop.fit, 'cover');
  const left = crop.x / 100 * .5;
  assert.ok(left <= .7 && left + .5 >= .95);
  assert.equal(crop.y, 50);
});
test('widely separated faces fall back to the whole photo', () => {
  const photo = {width: 2000, height: 1000, face_bounds: {x: .05, y: .2, width: .9, height: .4}};
  assert.equal(faceCrop(photo, 1000, 1000).fit, 'contain');
  assert.equal(faceCrop(photo, 2000, 1000).fit, 'cover');
});
test('portrait crops preserve faces near the top and bottom on resize', () => {
  for (const y of [0, .75]) {
    const photo = {width: 1000, height: 2000, face_bounds: {x: .2, y, width: .4, height: .25}};
    const crop = faceCrop(photo, 1000, 1000);
    const top = crop.y / 100 * .5;
    assert.ok(top <= y + 1e-9 && top + .5 >= y + .25 - 1e-9);
    assert.equal(faceCrop(photo, 2000, 500).fit, 'contain');
  }
});
test('missing or malformed face bounds use existing rendering', () => {
  assert.equal(faceCrop({width: 100, height: 100}, 100, 100), null);
  assert.equal(faceCrop({width: 100, height: 100, face_bounds: {x: -1, y: 0, width: .2, height: .2}}, 100, 100), null);
});


test('four landscapes stack on portrait screens and form a grid on landscape screens', () => {
  const photos = Array.from({length: 4}, () => ({width: 1600, height: 900}));
  assert.equal(chooseLayout(photos, 1080, 1920), 'quad-rows');
  assert.equal(chooseLayout(photos, 1920, 1080), 'quad-grid');
});

test('face bounds influence layout while wide face groups retain the whole image', () => {
  const photos = [{width: 1600, height: 900}, {width: 900, height: 1600}, {width: 1600, height: 900}];
  assert.equal(chooseLayout(photos, 1920, 1080), 'triple-lead-top');
  photos[2].face_bounds = {x: .4, y: .4, width: .2, height: .2};
  assert.equal(chooseLayout(photos, 1920, 1080), 'triple-cols');
  photos[2].face_bounds = {x: .05, y: .05, width: .9, height: .9};
  assert.equal(chooseLayout(photos, 1920, 1080), 'triple-lead-top');
});
