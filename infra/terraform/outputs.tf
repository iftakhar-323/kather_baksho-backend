output "public_ip" {
  description = "Public Elastic IP assigned to the EC2 instance"
  value       = aws_eip.app_eip.public_ip
}

output "traefik_gateway_url" {
  description = "Public URL for Traefik Cloud-Native Ingress Gateway"
  value       = "http://${aws_eip.app_eip.public_ip}:8085"
}

output "web_storefront_url" {
  description = "Public URL for React Storefront (Traefik Ingress)"
  value       = "http://${aws_eip.app_eip.public_ip}:8085"
}

output "traefik_dashboard_url" {
  description = "Public URL for Traefik Live Observability Dashboard"
  value       = "http://${aws_eip.app_eip.public_ip}:8086/dashboard/"
}

output "grafana_url" {
  description = "Public URL for Grafana Enterprise Telemetry Dashboard"
  value       = "http://${aws_eip.app_eip.public_ip}:3000"
}

output "ssh_connection_command" {
  description = "SSH administrative login command"
  value       = "ssh -i ~/.ssh/id_rsa ubuntu@${aws_eip.app_eip.public_ip}"
}

