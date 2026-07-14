--TEST--
P0 alternative if syntax
--FILE--
<?php
$value = 2;
if ($value === 1):
    echo 'one';
elseif ($value === 2):
    echo 'two';
else:
    echo 'other';
endif;
echo ':';
if (true):
    if (false) {
        echo 'bad';
    } else {
        echo 'nested';
    }
endif;
?>
--EXPECT--
two:nested
