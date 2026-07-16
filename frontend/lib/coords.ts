// Yandex Maps uses [longitude, latitude]; our component props use [lat, lon]
// (the order used across our map/geocode interfaces). Convert at the boundary.
export function toLngLat(latLon: [number, number]): [number, number] {
  return [latLon[1], latLon[0]];
}
