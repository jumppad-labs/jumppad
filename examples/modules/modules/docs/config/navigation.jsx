export const navigation = [
  {
    title: 'Index',
    links: [
      { title: 'Index', href: '/' },
    ]
  },
  {
    title: 'Install terraform',
    links: [
      { title: 'Manual installation', href: '/terraform_basics/installation/manual_installation' },
      { title: 'Verify installation', href: '/terraform_basics/installation/verify_installation' },
      { title: 'Terraform version', href: '/terraform_basics/installation/terraform_version' },
    ]
  },
  {
    title: 'Terraform workflow',
    links: [
      { title: 'Terraform init', href: '/terraform_basics/workflow/terraform_init' },
      { title: 'Terraform plan', href: '/terraform_basics/workflow/terraform_plan' },
      { title: 'Terraform apply', href: '/terraform_basics/workflow/terraform_apply' },
      { title: 'Update resources', href: '/terraform_basics/workflow/update_resources' },
      { title: 'Terraform destroy', href: '/terraform_basics/workflow/terraform_destroy' },
    ]
  },
  {
    title: 'Providers',
    links: [
      { title: 'Find and install providers', href: '/terraform_basics/providers/install_provider' },
      { title: 'Configuration', href: '/terraform_basics/providers/provider_configuration' },
    ]
  },
  {
    title: 'State',
    links: [
      { title: 'Viewing state', href: '/terraform_basics/state/viewing_state' },
      { title: 'List state', href: '/terraform_basics/state/list_state' },
      { title: 'Show state', href: '/terraform_basics/state/show_state' },
    ]
  }
]