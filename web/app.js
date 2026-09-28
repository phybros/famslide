(() => {
  const layers = [document.getElementById('scene-a'), document.getElementById('scene-b')];
  const empty = document.getElementById('empty');
  const duration = 20000;
  const fade = 1500;
  const motions = ['motion-in', 'motion-out', 'motion-pan-x', 'motion-pan-y', 'motion-burns'];
  const chooseLayout = photos => FamslideLayout.chooseLayout(photos, innerWidth, innerHeight);
  let manifest = null;
  let pending = null;
  let playlist = [];
  let index = 0;
  let active = 0;
  let timer = null;
  let prepared = null;
  let failures = 0;
  let fetching = false;

  const wait = ms => new Promise(resolve => setTimeout(resolve, ms));

  function photosFor(scene) {
    if (Array.isArray(scene.photos) && scene.photos.length) return scene.photos;
    if (scene.image) return [{url: scene.image, width: innerWidth, height: innerHeight}];
    return [];
  }

  function applyLayout(layer) {
    if (!layer.scene) return;
    const photos = photosFor(layer.scene);
    layer.className = `scene-layer layout-${chooseLayout(photos)}` + (layer.classList.contains('visible') ? ' visible' : '');
  }

  function imageReady(image) {
    return new Promise((resolve, reject) => {
      if (image.complete) return image.naturalWidth ? resolve() : reject(new Error('image unavailable'));
      image.onload = resolve;
      image.onerror = () => reject(new Error('image unavailable'));
    }).then(() => image.decode ? image.decode() : undefined);
  }

  function prepare(scene, layer) {
    layer.replaceChildren();
    layer.scene = scene;
    const images = [];
    photosFor(scene).forEach((photo, i) => {
      const tile = document.createElement('div');
      tile.className = 'tile';
      tile.dataset.motion = motions[Math.floor(Math.random() * motions.length)];
      const image = document.createElement('img');
      image.alt = '';
      image.draggable = false;
      image.decoding = 'async';
      image.src = photo.url;
      tile.append(image);
      layer.append(tile);
      images.push(image);
    });
    applyLayout(layer);
    const ready = images.length ? Promise.all(images.map(imageReady)) : Promise.reject(new Error('empty scene'));
    ready.catch(() => {});
    return {id: scene.id, ready};
  }

  function prepareNext() {
    if (!playlist.length) { prepared = null; return; }
    prepared = prepare(playlist[index], layers[1 - active]);
  }

  function advance() {
    if (pending) {
      manifest = pending;
      pending = null;
      playlist = FamslidePlaylist.buildPlaylist(manifest, playlist);
      index = 0;
    } else if (++index >= playlist.length) {
      playlist = FamslidePlaylist.buildPlaylist(manifest, playlist);
      index = 0;
    }
  }

  async function showNext() {
    clearTimeout(timer);
    if (!playlist.length) {
      empty.classList.remove('hidden');
      timer = setTimeout(refresh, 10000);
      return;
    }
    const scene = playlist[index];
    const target = layers[1 - active];
    try {
      if (!prepared || prepared.id !== scene.id) prepared = prepare(scene, target);
      await prepared.ready;
      failures = 0;
      empty.classList.add('hidden');
      target.querySelectorAll('.tile').forEach(tile => tile.classList.add(tile.dataset.motion));
      target.classList.add('visible');
      layers[active].classList.remove('visible');
      await wait(fade);
      layers[active].replaceChildren();
      layers[active].scene = null;
      layers[active].className = 'scene-layer';
      active = 1 - active;
      advance();
      prepareNext();
      timer = setTimeout(showNext, duration - fade);
    } catch (_) {
      target.replaceChildren();
      target.scene = null;
      prepared = null;
      failures++;
      if (failures >= playlist.length) {
        empty.classList.remove('hidden');
        failures = 0;
        if (pending) { manifest = pending; pending = null; }
        playlist = FamslidePlaylist.buildPlaylist(manifest, playlist);
        index = 0;
        timer = setTimeout(showNext, 10000);
      } else {
        advance();
        timer = setTimeout(showNext, 300);
      }
    }
  }

  async function refresh() {
    if (fetching) return;
    fetching = true;
    try {
      const response = await fetch('/api/scenes', {cache: 'no-store'});
      if (!response.ok) throw new Error('manifest unavailable');
      const next = await response.json();
      next.scenes ||= [];
      if (!playlist.length) {
        manifest = next;
        playlist = FamslidePlaylist.buildPlaylist(next);
        index = 0;
        if (playlist.length) showNext();
      } else if (next.version !== manifest.version) {
        pending = next;
      }
    } catch (_) { /* Keep playing the current manifest. */ }
    finally { fetching = false; }
  }

  let resizeQueued = false;
  window.addEventListener('resize', () => {
    if (resizeQueued) return;
    resizeQueued = true;
    requestAnimationFrame(() => { layers.forEach(applyLayout); resizeQueued = false; });
  });
  let cursorTimer;
  window.addEventListener('pointermove', () => {
    document.body.classList.add('show-cursor');
    clearTimeout(cursorTimer);
    cursorTimer = setTimeout(() => document.body.classList.remove('show-cursor'), 2500);
  });
  window.addEventListener('keydown', event => { if (event.key.toLowerCase() === 'a') location.href = '/admin'; });
  refresh();
  setInterval(refresh, 60000);
  setInterval(() => { if (!playlist.length) refresh(); }, 10000);
})();
