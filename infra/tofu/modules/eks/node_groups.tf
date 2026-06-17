resource "aws_eks_node_group" "this" {
  for_each = var.node_groups

  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "${var.project}-${var.env}-${each.key}"
  node_role_arn   = aws_iam_role.nodes.arn
  subnet_ids      = var.private_subnets
  instance_types  = each.value.instance_types
  disk_size       = each.value.disk_size_gb

  scaling_config {
    desired_size = each.value.desired
    min_size     = each.value.min
    max_size     = each.value.max
  }

  update_config {
    max_unavailable_percentage = 25
  }

  dynamic "taint" {
    for_each = each.key == "spot" ? [1] : []
    content {
      key    = "node-role"
      value  = "spot"
      effect = "NO_SCHEDULE"
    }
  }

  labels = {
    Environment = var.env
    NodeGroup   = each.key
  }

  depends_on = [
    aws_iam_role_policy_attachment.nodes_worker,
    aws_iam_role_policy_attachment.nodes_cni,
    aws_iam_role_policy_attachment.nodes_ecr,
  ]

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }
}
