resource "mapbox_token" "example" {
  username     = "example"
  note         = "example"
  scopes       = ["styles:read", "fonts:read"]
  allowed_urls = ["https://docs.mapbox.com"]
}
