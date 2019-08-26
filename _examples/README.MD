# Examples Directory 

> Note: Each `tf` file in the examples directory is tested as a single file. Do not use multi-file TF configurations. 

## Basics 

- [Create a basic ingress](./ingress_basic/basic_ingress.tf)
- [Create a basic service](./service/basic_service.tf) 


## Tests and Bugs

- [Fix for 'got string expected struct'](./ingress_complex/complex_ingress.tf) this also provides a sample of using both the `kubectl` provider and the `kubernetes` provider in the same TF config. 