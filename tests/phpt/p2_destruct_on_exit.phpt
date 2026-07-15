--TEST--
P2 __destruct runs after exit
--FILE--
<?php
class Box {
    function __destruct() {
        echo " destruct";
    }
}

$box = new Box();
echo "body";
exit(" exit");
?>
--EXPECT--
body exit destruct
