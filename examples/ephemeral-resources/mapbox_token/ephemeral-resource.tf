# Keep a public token's value out of state and read it only when needed.
resource "mapbox_token" "example" {
  username    = "example"
  note        = "example"
  scopes      = ["styles:read", "fonts:read"]
  store_token = false
}

ephemeral "mapbox_token" "example" {
  username = mapbox_token.example.username
  # The resource id has the form "<token_id>:<username>".
  token_id = split(":", mapbox_token.example.id)[0]
}
