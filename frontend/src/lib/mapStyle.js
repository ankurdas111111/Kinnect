/**
 * Map style configuration for MapLibre GL.
 *
 * Primary: OpenFreeMap vector tiles — completely free, no API key,
 * sharp GPU-rendered tiles, excellent OSM coverage in India, global CDN.
 *
 * Fallback: CARTO raster tiles if vector style fails to load.
 */

// Liberty style JSON is bundled locally (frontend/public/map-style.json) to
// eliminate one external round-trip at map init time and enable offline use.
// Update the file periodically: curl -s https://tiles.openfreemap.org/styles/liberty -o frontend/public/map-style.json
export const MAP_STYLE = '/map-style.json';

// Raster fallback if OpenFreeMap is unreachable
export const RASTER_STYLE = {
  version: 8,
  sources: {
    osm: {
      type: 'raster',
      tiles: [
        'https://a.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}.png',
        'https://b.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}.png',
        'https://c.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}.png'
      ],
      tileSize: 256,
      maxzoom: 18,
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
    }
  },
  layers: [{ id: 'osm-tiles', type: 'raster', source: 'osm' }]
};
