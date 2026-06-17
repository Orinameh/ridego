locals {
  addons = {
    vpc-cni            = "v1.18.1-eksbuild.1"
    coredns            = "v1.11.1-eksbuild.8"
    kube-proxy         = "v1.30.0-eksbuild.3"
    aws-ebs-csi-driver = "v1.32.0-eksbuild.1"
  }
}

resource "aws_eks_addon" "this" {
  for_each = local.addons

  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = each.key
  addon_version               = each.value
  resolve_conflicts_on_update = "OVERWRITE"

  depends_on = [aws_eks_node_group.this]
}
