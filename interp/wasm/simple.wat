(module
  (memory 1)
  (export "memory" (memory 0))
  
  (func $solve (param $input_ptr i32) (param $input_size i32) (result i32 i32)
    ;; Return hardcoded response: pointer 0, size 0
    (i32.const 0)
    (i32.const 0)
  )
  
  (export "solve" (func $solve))
)
