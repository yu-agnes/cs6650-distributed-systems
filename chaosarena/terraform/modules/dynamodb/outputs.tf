output "albums_table_name" {
  value = aws_dynamodb_table.albums.name
}

output "photos_table_name" {
  value = aws_dynamodb_table.photos.name
}
