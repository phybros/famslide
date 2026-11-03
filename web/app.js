(() => {
  const layers = [document.getElementById('scene-a'), document.getElementById('scene-b')];
  const empty = document.getElementById('empty');
  let duration = 20000;
  const buildPlaylist = (source, previous = []) => FamslidePlaylist.buildPlaylist(source, previous, Math.random, {width: innerWidth, height: innerHeight});
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

  let generation = 0;

  function photosFor(scene) {
    if (Array.isArray(scene.photos) && scene.photos.length) return scene.photos;
    if (scene.image) return [{url: scene.image, width: innerWidth, height: innerHeight}];
    return [];
  }

  function applyLayout(layer) {
    if (!layer.scene) return;
    const photos = photosFor(layer.scene);
    layer.className = `scene-layer layout-${chooseLayout(photos)}` + (layer.classList.contains('visible') ? ' visible' : '');
    layer.querySelectorAll('.tile').forEach((tile, i) => {
      const crop = FamslideLayout.faceCrop(photos[i], tile.clientWidth, tile.clientHeight);
      const image = tile.querySelector('img');
      image.style.objectFit = crop?.fit || 'cover';
      image.style.objectPosition = crop ? `${crop.x}% ${crop.y}%` : 'center';
    });
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
    photosFor(scene).forEach(photo => {
      const tile = document.createElement('div');
      tile.className = 'tile';
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
      playlist = buildPlaylist(manifest, playlist);
      index = 0;
    } else if (++index >= playlist.length) {
      playlist = buildPlaylist(manifest, playlist);
      index = 0;
    }
  }

  async function showNext() {
    const current = ++generation;
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
      if (current !== generation) return;
      failures = 0;
      empty.classList.add('hidden');
      target.classList.add('visible');
      layers[active].classList.remove('visible');
      layers[active].replaceChildren();
      layers[active].scene = null;
      layers[active].className = 'scene-layer';
      active = 1 - active;
      advance();
      prepareNext();
      timer = setTimeout(showNext, duration);
    } catch (_) {
      if (current !== generation) return;
      target.replaceChildren();
      target.scene = null;
      prepared = null;
      failures++;
      if (failures >= playlist.length) {
        empty.classList.remove('hidden');
        failures = 0;
        if (pending) { manifest = pending; pending = null; }
        playlist = buildPlaylist(manifest, playlist);
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
      if (Number.isFinite(next.slideshow_interval_ms) && next.slideshow_interval_ms >= 2000 && next.slideshow_interval_ms <= 86400000) {
        duration = next.slideshow_interval_ms;
      }
      if (!playlist.length) {
        manifest = next;
        playlist = buildPlaylist(next);
        index = 0;
        if (playlist.length) showNext();
      } else if (next.version !== manifest.version) {
        pending = next;
      }
    } catch (_) { /* Keep playing the current manifest. */ }
    finally { fetching = false; }
  }

  let resizeQueued = false;
  let landscape = innerWidth >= innerHeight;
  window.addEventListener('resize', () => {
    if (resizeQueued) return;
    resizeQueued = true;
    requestAnimationFrame(() => {
      resizeQueued = false;
      const nextLandscape = innerWidth >= innerHeight;
      if (manifest && nextLandscape !== landscape) {
        if (pending) { manifest = pending; pending = null; }
        playlist = buildPlaylist(manifest, playlist);
        index = 0;
        prepared = null;
        showNext();
      } else layers.forEach(applyLayout);
      landscape = nextLandscape;
    });
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
