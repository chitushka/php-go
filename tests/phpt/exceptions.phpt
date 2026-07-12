--TEST--
throw and catch exception
--FILE--
<?php
class Worker {
    function run() {
        throw new Exception('failed');
    }
}

try {
    $worker = new Worker();
    $worker->run();
    echo 'not reached';
} catch (Exception $e) {
    echo $e->getMessage();
}
--EXPECT--
failed
