--TEST--
P2 magic __callStatic
--FILE--
<?php
class Dynamic {
    private static function hidden() {
        return 'no';
    }

    public static function __callStatic($name, $args) {
        echo $name;
        foreach ($args as $arg) {
            echo ':' . $arg;
        }
        echo "\n";
    }
}

Dynamic::missing(1, 'x');
Dynamic::hidden('secret');
$method = 'dynamic';
Dynamic::$method('call');
?>
--EXPECT--
missing:1:x
hidden:secret
dynamic:call
