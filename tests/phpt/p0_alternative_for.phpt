--TEST--
P0 alternative for syntax
--FILE--
<?php
for ($i = 0; $i < 5; $i++):
    if ($i === 1):
        continue;
    endif;
    if ($i === 4):
        break;
    endif;
    echo $i;
endfor;
echo ':';
echo $i;
?>
--EXPECT--
023:4
