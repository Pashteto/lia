// Yandex Maps uses [longitude, latitude]; our component props use [lat, lon]
// (Leaflet order). Convert at the boundary.
export function toLngLat(latLon: [number, number]): [number, number] {
  return [latLon[1], latLon[0]];
}
