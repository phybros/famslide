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
