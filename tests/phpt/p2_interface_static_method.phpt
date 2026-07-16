--TEST--
P2 static interface method implementation
--FILE--
<?php
interface StaticNamed {
    public static function name();
}

class StaticUser implements StaticNamed {
    public static function name() {
        return 'Ada';
    }
}

echo StaticUser::name();
?>
--EXPECT--
Ada
