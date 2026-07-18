--TEST--
P1 bracketed namespace blocks
--FILE--
<?php
namespace Lib {
    define('Lib\LOCAL_CONST', 'const');

    echo label();
    echo "\n";

    function label() {
        return __NAMESPACE__;
    }

    class Box {
        public function name() {
            return __NAMESPACE__;
        }
    }
}

namespace App {
    use Lib\Box as ImportedBox;

    function label() {
        return __NAMESPACE__;
    }

    $box = new ImportedBox();
    echo label();
    echo "\n";
    echo \Lib\label();
    echo "\n";
    echo $box->name();
    echo "\n";
    echo \Lib\LOCAL_CONST;
}

namespace {
    echo "\n";
    echo App\label();
    echo "\n";
    echo strlen('abcd');
    echo "\n";
    $box = new Lib\Box();
    echo $box instanceof Lib\Box ? 'instance' : 'bad';
}
?>
--EXPECT--
Lib
App
Lib
Lib
const
App
4
instance
