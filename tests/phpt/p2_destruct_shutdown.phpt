--TEST--
P2 __destruct runs at shutdown
--FILE--
<?php
class Box {
    function __destruct() {
        echo " destruct";
    }
}

$box = new Box();
echo "body";
?>
--EXPECT--
body destruct
